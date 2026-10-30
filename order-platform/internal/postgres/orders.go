package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/LeonardoCAlves/synechron/order-platform/internal/catalog"
	"github.com/LeonardoCAlves/synechron/order-platform/internal/events"
	"github.com/LeonardoCAlves/synechron/order-platform/internal/money"
	"github.com/LeonardoCAlves/synechron/order-platform/internal/orderapp"
	"github.com/LeonardoCAlves/synechron/order-platform/internal/orders"
)

type OrderStore struct {
	db *sql.DB
}

func NewOrderStore(db *sql.DB) *OrderStore {
	return &OrderStore{db: db}
}

func (store *OrderStore) FindOrder(ctx context.Context, id string) (orders.Order, error) {
	order, err := loadOrder(ctx, store.db, id)
	if errors.Is(err, sql.ErrNoRows) {
		return orders.Order{}, orderapp.ErrOrderNotFound
	}
	if err != nil {
		return orders.Order{}, fmt.Errorf("find order: %w", err)
	}
	return order, nil
}

func (store *OrderStore) FindOrderByIdempotency(ctx context.Context, key, fingerprint string) (orders.Order, bool, error) {
	var storedFingerprint, orderID string
	err := store.db.QueryRowContext(ctx, `
		SELECT request_fingerprint, resource_id
		FROM idempotency_requests WHERE idempotency_key = $1`, key).Scan(&storedFingerprint, &orderID)
	if errors.Is(err, sql.ErrNoRows) {
		return orders.Order{}, false, nil
	}
	if err != nil {
		return orders.Order{}, false, fmt.Errorf("find order idempotency record: %w", err)
	}
	if storedFingerprint != fingerprint {
		return orders.Order{}, false, orderapp.ErrIdempotencyConflict
	}
	order, err := store.FindOrder(ctx, orderID)
	if err != nil {
		return orders.Order{}, false, fmt.Errorf("load idempotent order: %w", err)
	}
	return order, true, nil
}

func (store *OrderStore) CreateOrder(ctx context.Context, order orders.Order, key, fingerprint string) (orders.Order, error) {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return orders.Order{}, fmt.Errorf("begin order creation: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `
		INSERT INTO idempotency_requests(idempotency_key, request_fingerprint, resource_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (idempotency_key) DO NOTHING`, key, fingerprint, order.ID)
	if err != nil {
		return orders.Order{}, fmt.Errorf("reserve order idempotency key: %w", err)
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return orders.Order{}, fmt.Errorf("check order idempotency reservation: %w", err)
	}
	if inserted == 0 {
		var existingFingerprint, orderID string
		if err := tx.QueryRowContext(ctx, `
			SELECT request_fingerprint, resource_id
			FROM idempotency_requests WHERE idempotency_key = $1`, key).Scan(&existingFingerprint, &orderID); err != nil {
			return orders.Order{}, fmt.Errorf("load concurrent order idempotency record: %w", err)
		}
		if existingFingerprint != fingerprint {
			return orders.Order{}, orderapp.ErrIdempotencyConflict
		}
		existing, err := loadOrder(ctx, tx, orderID)
		if err != nil {
			return orders.Order{}, fmt.Errorf("load concurrent order result: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return orders.Order{}, fmt.Errorf("commit idempotent order lookup: %w", err)
		}
		return existing, nil
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO orders(id, total_cents, currency, status, created_at)
		VALUES ($1, $2, $3, $4, $5)`,
		order.ID, order.TotalCents, string(order.Currency), order.Status, order.CreatedAt); err != nil {
		return orders.Order{}, fmt.Errorf("insert order: %w", err)
	}
	for index, item := range order.Items {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO order_lines(order_id, line_number, product_id, product_name, quantity, unit_price_cents, currency)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			order.ID, index, item.ProductID, item.ProductName, item.Quantity, item.UnitPriceCents, string(item.Currency)); err != nil {
			return orders.Order{}, fmt.Errorf("insert order line %d: %w", index, err)
		}
	}
	event, err := newOrderCreatedEvent(order)
	if err != nil {
		return orders.Order{}, err
	}
	if err := insertOutboxEvent(ctx, tx, event); err != nil {
		return orders.Order{}, err
	}
	if err := tx.Commit(); err != nil {
		return orders.Order{}, fmt.Errorf("commit order and outbox event: %w", err)
	}
	return order, nil
}

func newOrderCreatedEvent(order orders.Order) (events.Envelope, error) {
	return events.New("order.created.v1", order.ID, 1, order.CreatedAt, order)
}

func (store *OrderStore) FindCatalogProduct(ctx context.Context, id string) (catalog.Product, error) {
	var product catalog.Product
	var currency string
	err := store.db.QueryRowContext(ctx, `
		SELECT id, name, price_cents, currency, active
		FROM catalog_product_projection WHERE id = $1`, id).Scan(
		&product.ID, &product.Name, &product.PriceCents, &currency, &product.Active)
	if errors.Is(err, sql.ErrNoRows) {
		return catalog.Product{}, orderapp.ErrCatalogProductNotFound
	}
	if err != nil {
		return catalog.Product{}, fmt.Errorf("find catalog projection product: %w", err)
	}
	product.Currency = money.Currency(currency)
	return product, nil
}

func (store *OrderStore) PutCatalogProduct(ctx context.Context, product catalog.Product) error {
	_, err := store.db.ExecContext(ctx, `
		INSERT INTO catalog_product_projection(id, name, price_cents, currency, active)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			price_cents = EXCLUDED.price_cents,
			currency = EXCLUDED.currency,
			active = EXCLUDED.active,
			updated_at = now()`,
		product.ID, product.Name, product.PriceCents, string(product.Currency), product.Active)
	if err != nil {
		return fmt.Errorf("upsert catalog projection product: %w", err)
	}
	return nil
}

type queryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func loadOrder(ctx context.Context, db queryer, id string) (orders.Order, error) {
	order, err := scanOrder(db.QueryRowContext(ctx, `
		SELECT id, total_cents, currency, status, created_at
		FROM orders WHERE id = $1`, id))
	if err != nil {
		return orders.Order{}, err
	}
	items, err := loadOrderLines(ctx, db, id)
	if err != nil {
		return orders.Order{}, err
	}
	order.Items = items
	return order, nil
}

func loadOrderLines(ctx context.Context, db queryer, orderID string) ([]orders.LineItem, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT product_id, product_name, quantity, unit_price_cents, currency
		FROM order_lines WHERE order_id = $1 ORDER BY line_number`, orderID)
	if err != nil {
		return nil, fmt.Errorf("query order lines: %w", err)
	}
	defer rows.Close()

	items := make([]orders.LineItem, 0)
	for rows.Next() {
		var item orders.LineItem
		var currency string
		if err := rows.Scan(&item.ProductID, &item.ProductName, &item.Quantity, &item.UnitPriceCents, &currency); err != nil {
			return nil, fmt.Errorf("scan order line: %w", err)
		}
		item.Currency = money.Currency(currency)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate order lines: %w", err)
	}
	return items, nil
}

func scanOrder(row rowScanner) (orders.Order, error) {
	var order orders.Order
	var currency string
	if err := row.Scan(&order.ID, &order.TotalCents, &currency, &order.Status, &order.CreatedAt); err != nil {
		return orders.Order{}, err
	}
	order.Currency = money.Currency(currency)
	order.CreatedAt = order.CreatedAt.UTC()
	return order, nil
}

var (
	_ orderapp.OrderRepository   = (*OrderStore)(nil)
	_ orderapp.CatalogProjection = (*OrderStore)(nil)
)
