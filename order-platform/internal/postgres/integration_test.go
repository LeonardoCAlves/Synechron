package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/LeonardoCAlves/synechron/order-platform/internal/catalog"
	"github.com/LeonardoCAlves/synechron/order-platform/internal/catalogapp"
	"github.com/LeonardoCAlves/synechron/order-platform/internal/events"
	"github.com/LeonardoCAlves/synechron/order-platform/internal/identity"
	"github.com/LeonardoCAlves/synechron/order-platform/internal/orderapp"
)

func TestCatalogPostgresPersistsIdempotencyAndTransactionalOutbox(t *testing.T) {
	db := openIsolatedDatabase(t, "ORDER_PLATFORM_CATALOG_TEST_DATABASE_URL", MigrateCatalog)
	ctx := context.Background()
	store := NewCatalogStore(db)
	service := catalogapp.NewService(store)
	input := catalogapp.CreateProductInput{Name: "Keyboard", Description: "Mechanical", PriceCents: 12999, Currency: "CAD"}

	product, err := service.CreateProduct(ctx, input, "catalog-test-1")
	if err != nil {
		t.Fatalf("CreateProduct() error = %v", err)
	}
	replayed, err := service.CreateProduct(ctx, input, "catalog-test-1")
	if err != nil {
		t.Fatalf("idempotent CreateProduct() error = %v", err)
	}
	if replayed.ID != product.ID {
		t.Fatalf("replayed product ID = %q, want %q", replayed.ID, product.ID)
	}
	if _, err := service.CreateProduct(ctx, catalogapp.CreateProductInput{
		Name: "Keyboard", Description: "Different", PriceCents: 12999, Currency: "CAD",
	}, "catalog-test-1"); !errors.Is(err, catalogapp.ErrIdempotencyConflict) {
		t.Fatalf("changed request error = %v, want idempotency conflict", err)
	}

	var eventType string
	var payload []byte
	if err := db.QueryRowContext(ctx, `SELECT event_type, payload FROM outbox_events WHERE aggregate_id = $1`, product.ID).Scan(&eventType, &payload); err != nil {
		t.Fatalf("read outbox event: %v", err)
	}
	var envelope events.Envelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		t.Fatalf("decode outbox event: %v", err)
	}
	if eventType != "catalog.product.created.v1" || envelope.Type != eventType || envelope.AggregateID != product.ID {
		t.Fatalf("unexpected outbox event: type=%q envelope=%#v", eventType, envelope)
	}

	if _, err := db.ExecContext(ctx, `
		ALTER TABLE outbox_events ADD CONSTRAINT reject_catalog_test_event
		CHECK (event_type <> 'catalog.product.created.v1' OR payload #>> '{data,name}' <> 'Mouse')`); err != nil {
		t.Fatalf("install outbox failure constraint: %v", err)
	}
	_, err = service.CreateProduct(ctx, catalogapp.CreateProductInput{
		Name: "Mouse", Description: "Wireless", PriceCents: 5999, Currency: "CAD",
	}, "catalog-test-rollback")
	if err == nil {
		t.Fatal("CreateProduct() succeeded despite outbox rejection")
	}
	var products, idempotencyRecords int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM catalog_products WHERE name = 'Mouse'`).Scan(&products); err != nil {
		t.Fatalf("count products after rollback: %v", err)
	}
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM idempotency_requests WHERE idempotency_key = 'catalog-test-rollback'`).Scan(&idempotencyRecords); err != nil {
		t.Fatalf("count idempotency records after rollback: %v", err)
	}
	if products != 0 || idempotencyRecords != 0 {
		t.Fatalf("failed outbox transaction persisted product=%d idempotency=%d", products, idempotencyRecords)
	}
}

func TestOrderPostgresPersistsSnapshotsIdempotencyAndTransactionalOutbox(t *testing.T) {
	db := openIsolatedDatabase(t, "ORDER_PLATFORM_ORDERS_TEST_DATABASE_URL", MigrateOrders)
	ctx := context.Background()
	store := NewOrderStore(db)
	product, err := catalog.NewProduct("projected-product-1", "Desk", "Wood", 10000, "CAD", time.Now())
	if err != nil {
		t.Fatalf("NewProduct() error = %v", err)
	}
	if err := store.PutCatalogProduct(ctx, product); err != nil {
		t.Fatalf("PutCatalogProduct() error = %v", err)
	}

	service := orderapp.NewService(store, store)
	input := orderapp.CreateOrderInput{Items: []orderapp.CreateOrderItemInput{{ProductID: product.ID, Quantity: 2}}}
	order, err := service.CreateOrder(ctx, input, "orders-test-1")
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}
	updatedProduct, err := catalog.NewProduct(product.ID, "Updated desk", "", 12000, "CAD", time.Now())
	if err != nil {
		t.Fatalf("updated NewProduct() error = %v", err)
	}
	if err := store.PutCatalogProduct(ctx, updatedProduct); err != nil {
		t.Fatalf("update catalog projection: %v", err)
	}
	replayed, err := service.CreateOrder(ctx, input, "orders-test-1")
	if err != nil {
		t.Fatalf("idempotent CreateOrder() error = %v", err)
	}
	if replayed.ID != order.ID || replayed.Items[0].ProductName != "Desk" || replayed.Items[0].UnitPriceCents != 10000 {
		t.Fatalf("replayed order did not preserve original snapshot: %#v", replayed)
	}
	if _, err := service.CreateOrder(ctx, orderapp.CreateOrderInput{
		Items: []orderapp.CreateOrderItemInput{{ProductID: product.ID, Quantity: 1}},
	}, "orders-test-1"); !errors.Is(err, orderapp.ErrIdempotencyConflict) {
		t.Fatalf("changed request error = %v, want idempotency conflict", err)
	}

	var eventType string
	var eventCount int
	if err := db.QueryRowContext(ctx, `SELECT event_type, count(*) FROM outbox_events GROUP BY event_type`).Scan(&eventType, &eventCount); err != nil {
		t.Fatalf("read order outbox event: %v", err)
	}
	if eventType != "order.created.v1" || eventCount != 1 {
		t.Fatalf("outbox event type/count = %q/%d, want order.created.v1/1", eventType, eventCount)
	}

	if _, err := db.ExecContext(ctx, `
		ALTER TABLE outbox_events ADD CONSTRAINT reject_order_test_event
		CHECK (event_type <> 'order.created.v1' OR payload #>> '{data,items,0,quantity}' <> '3')`); err != nil {
		t.Fatalf("install outbox failure constraint: %v", err)
	}
	_, err = service.CreateOrder(ctx, orderapp.CreateOrderInput{
		Items: []orderapp.CreateOrderItemInput{{ProductID: product.ID, Quantity: 3}},
	}, "orders-test-rollback")
	if err == nil {
		t.Fatal("CreateOrder() succeeded despite outbox rejection")
	}
	var ordersCount, idempotencyRecords int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM orders`).Scan(&ordersCount); err != nil {
		t.Fatalf("count orders after rollback: %v", err)
	}
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM idempotency_requests WHERE idempotency_key = 'orders-test-rollback'`).Scan(&idempotencyRecords); err != nil {
		t.Fatalf("count idempotency records after rollback: %v", err)
	}
	if ordersCount != 1 || idempotencyRecords != 0 {
		t.Fatalf("failed outbox transaction persisted orders=%d idempotency=%d", ordersCount, idempotencyRecords)
	}
}

func TestOrderPostgresInboxDeduplicatesAndIgnoresOutOfOrderEvents(t *testing.T) {
	db := openIsolatedDatabase(t, "ORDER_PLATFORM_ORDERS_TEST_DATABASE_URL", MigrateOrders)
	store := NewOrderStore(db)
	ctx := context.Background()

	newEvent := func(version int64, name string, occurredAt time.Time) events.Envelope {
		t.Helper()
		product, err := catalog.NewProduct("projected-product-versioned", name, "", version*100, "CAD", occurredAt)
		if err != nil {
			t.Fatalf("NewProduct() error = %v", err)
		}
		event, err := events.New("catalog.product.updated.v1", product.ID, 1, occurredAt, product)
		if err != nil {
			t.Fatalf("events.New() error = %v", err)
		}
		event.AggregateVersion = version
		return event
	}

	newer := newEvent(2, "New name", time.Now().UTC())
	if applied, err := store.ApplyCatalogEvent(ctx, newer); err != nil || !applied {
		t.Fatalf("ApplyCatalogEvent(newer) = %v, %v; want applied", applied, err)
	}
	if applied, err := store.ApplyCatalogEvent(ctx, newer); err != nil || applied {
		t.Fatalf("ApplyCatalogEvent(duplicate) = %v, %v; want duplicate", applied, err)
	}
	older := newEvent(1, "Old name", newer.OccurredAt.Add(-time.Minute))
	if applied, err := store.ApplyCatalogEvent(ctx, older); err != nil || !applied {
		t.Fatalf("ApplyCatalogEvent(older) = %v, %v; want inboxed event", applied, err)
	}

	product, err := store.FindCatalogProduct(ctx, newer.AggregateID)
	if err != nil {
		t.Fatalf("FindCatalogProduct() error = %v", err)
	}
	if product.Name != "New name" || product.PriceCents != 200 {
		t.Fatalf("stale event overwrote projection: %#v", product)
	}
	var inboxCount int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM inbox_events`).Scan(&inboxCount); err != nil {
		t.Fatalf("count inbox events: %v", err)
	}
	if inboxCount != 2 {
		t.Fatalf("inbox event count = %d, want 2 distinct events", inboxCount)
	}
}

func TestOutboxRetriesAndDeadLettersAfterBoundedAttempts(t *testing.T) {
	db := openIsolatedDatabase(t, "ORDER_PLATFORM_CATALOG_TEST_DATABASE_URL", MigrateCatalog)
	ctx := context.Background()
	store := NewCatalogStore(db)
	service := catalogapp.NewService(store)
	product, err := service.CreateProduct(ctx, catalogapp.CreateProductInput{
		Name: "Retry product", PriceCents: 500, Currency: "CAD",
	}, "outbox-retry-test")
	if err != nil {
		t.Fatalf("CreateProduct() error = %v", err)
	}

	outbox := NewOutboxStore(db)
	messages, err := outbox.ClaimPending(ctx, 10, time.Minute)
	if err != nil || len(messages) != 1 {
		t.Fatalf("ClaimPending() = %d messages, %v; want one message", len(messages), err)
	}
	if messages[0].AggregateID != product.ID || messages[0].Attempts != 0 {
		t.Fatalf("unexpected claimed event: %#v", messages[0])
	}
	deadLettered, err := outbox.RecordFailure(ctx, messages[0].ID, "temporary", time.Second, 2)
	if err != nil || deadLettered {
		t.Fatalf("first RecordFailure() = %v, %v; want retryable event", deadLettered, err)
	}

	if _, err := db.ExecContext(ctx, `UPDATE outbox_events SET next_attempt_at = now() WHERE id = $1`, messages[0].ID); err != nil {
		t.Fatalf("make retry available: %v", err)
	}
	messages, err = outbox.ClaimPending(ctx, 10, time.Minute)
	if err != nil || len(messages) != 1 || messages[0].Attempts != 1 {
		t.Fatalf("ClaimPending(retry) = %#v, %v; want one event after one attempt", messages, err)
	}
	deadLettered, err = outbox.RecordFailure(ctx, messages[0].ID, "permanent", 0, 2)
	if err != nil || !deadLettered {
		t.Fatalf("second RecordFailure() = %v, %v; want dead-lettered event", deadLettered, err)
	}
	messages, err = outbox.ClaimPending(ctx, 10, time.Minute)
	if err != nil || len(messages) != 0 {
		t.Fatalf("ClaimPending(dead-lettered) = %#v, %v; want no events", messages, err)
	}
}

func TestOutboxPreservesAggregateVersionOrder(t *testing.T) {
	db := openIsolatedDatabase(t, "ORDER_PLATFORM_CATALOG_TEST_DATABASE_URL", MigrateCatalog)
	ctx := context.Background()
	store := NewCatalogStore(db)
	product, err := catalogapp.NewService(store).CreateProduct(ctx, catalogapp.CreateProductInput{
		Name: "Versioned product", PriceCents: 500, Currency: "CAD",
	}, "outbox-order-test")
	if err != nil {
		t.Fatalf("CreateProduct() error = %v", err)
	}

	var firstEventID string
	if err := db.QueryRowContext(ctx, `
		SELECT id FROM outbox_events WHERE aggregate_id = $1 AND aggregate_version = 1`,
		product.ID).Scan(&firstEventID); err != nil {
		t.Fatalf("find initial outbox event: %v", err)
	}
	if err := storeOutboxVersion(ctx, db, product, 2); err != nil {
		t.Fatalf("store aggregate version 2: %v", err)
	}
	if err := storeOutboxVersion(ctx, db, product, 3); err != nil {
		t.Fatalf("store aggregate version 3: %v", err)
	}

	outbox := NewOutboxStore(db)
	if err := outbox.MarkPublished(ctx, firstEventID); err != nil {
		t.Fatalf("mark initial event published: %v", err)
	}
	messages, err := outbox.ClaimPending(ctx, 10, time.Minute)
	if err != nil || len(messages) != 1 || messages[0].AggregateVersion != 2 {
		t.Fatalf("ClaimPending() = %#v, %v; want only aggregate version 2", messages, err)
	}
	deadLettered, err := outbox.RecordFailure(ctx, messages[0].ID, "requires recovery", 0, 1)
	if err != nil || !deadLettered {
		t.Fatalf("dead-letter aggregate version 2 = %v, %v; want dead-lettered", deadLettered, err)
	}
	messages, err = outbox.ClaimPending(ctx, 10, time.Minute)
	if err != nil || len(messages) != 0 {
		t.Fatalf("ClaimPending() after dead-letter = %#v, %v; want later versions blocked", messages, err)
	}

	if _, err := db.ExecContext(ctx, `
		UPDATE outbox_events
		SET dead_lettered_at = NULL, attempts = 0, next_attempt_at = now()
		WHERE aggregate_id = $1 AND aggregate_version = 2`, product.ID); err != nil {
		t.Fatalf("requeue aggregate version 2: %v", err)
	}
	messages, err = outbox.ClaimPending(ctx, 10, time.Minute)
	if err != nil || len(messages) != 1 || messages[0].AggregateVersion != 2 {
		t.Fatalf("ClaimPending() after recovery = %#v, %v; want aggregate version 2", messages, err)
	}
	if err := outbox.MarkPublished(ctx, messages[0].ID); err != nil {
		t.Fatalf("mark aggregate version 2 published: %v", err)
	}
	messages, err = outbox.ClaimPending(ctx, 10, time.Minute)
	if err != nil || len(messages) != 1 || messages[0].AggregateVersion != 3 {
		t.Fatalf("ClaimPending() after version 2 = %#v, %v; want aggregate version 3", messages, err)
	}
}

func storeOutboxVersion(ctx context.Context, db *sql.DB, product catalog.Product, version int64) error {
	event, err := events.New("catalog.product.updated.v1", product.ID, 1, time.Now().UTC(), product)
	if err != nil {
		return fmt.Errorf("create aggregate version %d event: %w", version, err)
	}
	event.AggregateVersion = version
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin aggregate version %d outbox transaction: %w", version, err)
	}
	defer tx.Rollback()
	if err := insertOutboxEvent(ctx, tx, event); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit aggregate version %d outbox event: %w", version, err)
	}
	return nil
}

func openIsolatedDatabase(t *testing.T, envName string, migrate func(context.Context, *sql.DB) error) *sql.DB {
	t.Helper()
	dataSourceName := os.Getenv(envName)
	if strings.TrimSpace(dataSourceName) == "" {
		t.Skipf("set %s to a disposable PostgreSQL database to run this integration test", envName)
	}

	admin, err := Open(context.Background(), dataSourceName)
	if err != nil {
		t.Fatalf("connect to integration database: %v", err)
	}
	identifier, err := newSchemaIdentifier()
	if err != nil {
		admin.Close()
		t.Fatalf("create test schema identifier: %v", err)
	}
	schema := "order_platform_test_" + identifier
	if _, err := admin.ExecContext(context.Background(), `CREATE SCHEMA "`+schema+`"`); err != nil {
		admin.Close()
		t.Fatalf("create isolated integration schema: %v", err)
	}

	db, err := Open(context.Background(), dataSourceName)
	if err != nil {
		_, _ = admin.ExecContext(context.Background(), `DROP SCHEMA "`+schema+`" CASCADE`)
		admin.Close()
		t.Fatalf("open integration schema connection: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if _, err := db.ExecContext(context.Background(), `SET search_path TO "`+schema+`"`); err != nil {
		db.Close()
		_, _ = admin.ExecContext(context.Background(), `DROP SCHEMA "`+schema+`" CASCADE`)
		admin.Close()
		t.Fatalf("set isolated integration schema: %v", err)
	}
	if err := migrate(context.Background(), db); err != nil {
		db.Close()
		_, _ = admin.ExecContext(context.Background(), `DROP SCHEMA "`+schema+`" CASCADE`)
		admin.Close()
		t.Fatalf("migrate isolated integration schema: %v", err)
	}
	if err := migrate(context.Background(), db); err != nil {
		db.Close()
		_, _ = admin.ExecContext(context.Background(), `DROP SCHEMA "`+schema+`" CASCADE`)
		admin.Close()
		t.Fatalf("repeat isolated integration migration: %v", err)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close integration database: %v", err)
		}
		if _, err := admin.ExecContext(context.Background(), `DROP SCHEMA "`+schema+`" CASCADE`); err != nil {
			t.Errorf("drop isolated integration schema: %v", err)
		}
		if err := admin.Close(); err != nil {
			t.Errorf("close integration database administrator: %v", err)
		}
	})
	return db
}

func newSchemaIdentifier() (string, error) {
	id, err := identity.NewUUID()
	if err != nil {
		return "", fmt.Errorf("generate schema identifier: %w", err)
	}
	return strings.ReplaceAll(id, "-", ""), nil
}
