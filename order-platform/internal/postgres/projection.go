package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/LeonardoCAlves/synechron/order-platform/internal/catalog"
	"github.com/LeonardoCAlves/synechron/order-platform/internal/events"
)

var ErrPoisonCatalogEvent = errors.New("invalid or unsupported catalog event")

func (store *OrderStore) ApplyCatalogEvent(ctx context.Context, event events.Envelope) (bool, error) {
	if err := event.Validate(); err != nil {
		return false, fmt.Errorf("%w: %v", ErrPoisonCatalogEvent, err)
	}
	if (event.Type != "catalog.product.created.v1" && event.Type != "catalog.product.updated.v1") ||
		event.Version != 1 || event.AggregateVersion < 1 {
		return false, fmt.Errorf("%w: unsupported type or version", ErrPoisonCatalogEvent)
	}

	var product catalog.Product
	if err := json.Unmarshal(event.Data, &product); err != nil {
		return false, fmt.Errorf("%w: decode product payload: %v", ErrPoisonCatalogEvent, err)
	}
	if product.ID != event.AggregateID {
		return false, fmt.Errorf("%w: aggregate ID does not match product ID", ErrPoisonCatalogEvent)
	}
	validated, err := catalog.NewProduct(
		product.ID, product.Name, product.Description, product.PriceCents,
		string(product.Currency), product.CreatedAt,
	)
	if err != nil {
		return false, fmt.Errorf("%w: validate product payload: %v", ErrPoisonCatalogEvent, err)
	}
	validated.Active = product.Active

	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin catalog projection transaction: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `
		INSERT INTO inbox_events(event_id, aggregate_id, aggregate_version, event_type)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (event_id) DO NOTHING`,
		event.ID, event.AggregateID, event.AggregateVersion, event.Type)
	if err != nil {
		return false, fmt.Errorf("record catalog event in inbox: %w", err)
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("check catalog inbox insert: %w", err)
	}
	if inserted == 0 {
		if err := tx.Commit(); err != nil {
			return false, fmt.Errorf("commit duplicate catalog event: %w", err)
		}
		return false, nil
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO catalog_product_projection(
			id, name, price_cents, currency, active, aggregate_version, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			price_cents = EXCLUDED.price_cents,
			currency = EXCLUDED.currency,
			active = EXCLUDED.active,
			aggregate_version = EXCLUDED.aggregate_version,
			updated_at = EXCLUDED.updated_at
		WHERE catalog_product_projection.aggregate_version < EXCLUDED.aggregate_version`,
		validated.ID, validated.Name, validated.PriceCents, string(validated.Currency),
		validated.Active, event.AggregateVersion, event.OccurredAt)
	if err != nil {
		return false, fmt.Errorf("update catalog product projection: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit catalog inbox and projection: %w", err)
	}
	return true, nil
}
