package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/LeonardoCAlves/synechron/order-platform/internal/catalog"
	"github.com/LeonardoCAlves/synechron/order-platform/internal/catalogapp"
	"github.com/LeonardoCAlves/synechron/order-platform/internal/events"
	"github.com/LeonardoCAlves/synechron/order-platform/internal/money"
)

type CatalogStore struct {
	db *sql.DB
}

func NewCatalogStore(db *sql.DB) *CatalogStore {
	return &CatalogStore{db: db}
}

func (store *CatalogStore) Find(ctx context.Context, id string) (catalog.Product, error) {
	product, err := scanProduct(store.db.QueryRowContext(ctx, `
		SELECT id, name, description, price_cents, currency, active, created_at
		FROM catalog_products WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return catalog.Product{}, catalogapp.ErrProductNotFound
	}
	if err != nil {
		return catalog.Product{}, fmt.Errorf("find product: %w", err)
	}
	return product, nil
}

func (store *CatalogStore) FindByIdempotency(ctx context.Context, key, fingerprint string) (catalog.Product, bool, error) {
	var storedFingerprint, productID string
	err := store.db.QueryRowContext(ctx, `
		SELECT request_fingerprint, resource_id
		FROM idempotency_requests WHERE idempotency_key = $1`, key).Scan(&storedFingerprint, &productID)
	if errors.Is(err, sql.ErrNoRows) {
		return catalog.Product{}, false, nil
	}
	if err != nil {
		return catalog.Product{}, false, fmt.Errorf("find product idempotency record: %w", err)
	}
	if storedFingerprint != fingerprint {
		return catalog.Product{}, false, catalogapp.ErrIdempotencyConflict
	}
	product, err := store.Find(ctx, productID)
	if err != nil {
		return catalog.Product{}, false, fmt.Errorf("load idempotent product: %w", err)
	}
	return product, true, nil
}

func (store *CatalogStore) Create(ctx context.Context, product catalog.Product, key, fingerprint string) (catalog.Product, error) {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return catalog.Product{}, fmt.Errorf("begin product creation: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `
		INSERT INTO idempotency_requests(idempotency_key, request_fingerprint, resource_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (idempotency_key) DO NOTHING`, key, fingerprint, product.ID)
	if err != nil {
		return catalog.Product{}, fmt.Errorf("reserve product idempotency key: %w", err)
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return catalog.Product{}, fmt.Errorf("check product idempotency reservation: %w", err)
	}
	if inserted == 0 {
		var existingFingerprint, productID string
		if err := tx.QueryRowContext(ctx, `
			SELECT request_fingerprint, resource_id
			FROM idempotency_requests WHERE idempotency_key = $1`, key).Scan(&existingFingerprint, &productID); err != nil {
			return catalog.Product{}, fmt.Errorf("load concurrent product idempotency record: %w", err)
		}
		if existingFingerprint != fingerprint {
			return catalog.Product{}, catalogapp.ErrIdempotencyConflict
		}
		existing, err := scanProduct(tx.QueryRowContext(ctx, `
			SELECT id, name, description, price_cents, currency, active, created_at
			FROM catalog_products WHERE id = $1`, productID))
		if err != nil {
			return catalog.Product{}, fmt.Errorf("load concurrent product result: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return catalog.Product{}, fmt.Errorf("commit idempotent product lookup: %w", err)
		}
		return existing, nil
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO catalog_products(id, name, description, price_cents, currency, active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		product.ID, product.Name, product.Description, product.PriceCents, string(product.Currency), product.Active, product.CreatedAt); err != nil {
		return catalog.Product{}, fmt.Errorf("insert product: %w", err)
	}

	event, err := events.New("catalog.product.created.v1", product.ID, 1, product.CreatedAt, product)
	if err != nil {
		return catalog.Product{}, fmt.Errorf("build product-created event: %w", err)
	}
	if err := insertOutboxEvent(ctx, tx, event); err != nil {
		return catalog.Product{}, err
	}
	if err := tx.Commit(); err != nil {
		return catalog.Product{}, fmt.Errorf("commit product and outbox event: %w", err)
	}
	return product, nil
}

func insertOutboxEvent(ctx context.Context, tx *sql.Tx, event events.Envelope) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal outbox event: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO outbox_events(id, event_type, aggregate_id, schema_version, occurred_at, payload)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		event.ID, event.Type, event.AggregateID, event.Version, event.OccurredAt, string(payload)); err != nil {
		return fmt.Errorf("insert outbox event: %w", err)
	}
	return nil
}

type rowScanner interface {
	Scan(...any) error
}

func scanProduct(row rowScanner) (catalog.Product, error) {
	var product catalog.Product
	var currency string
	if err := row.Scan(&product.ID, &product.Name, &product.Description, &product.PriceCents, &currency, &product.Active, &product.CreatedAt); err != nil {
		return catalog.Product{}, err
	}
	product.Currency = money.Currency(currency)
	product.CreatedAt = product.CreatedAt.UTC()
	return product, nil
}
