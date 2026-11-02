package memory

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/LeonardoCAlves/synechron/order-platform/internal/catalog"
	"github.com/LeonardoCAlves/synechron/order-platform/internal/catalogapp"
	"github.com/LeonardoCAlves/synechron/order-platform/internal/orderapp"
	"github.com/LeonardoCAlves/synechron/order-platform/internal/orders"
)

func TestStoreProductIdempotencyAndDuplicateIDs(t *testing.T) {
	ctx := context.Background()
	store := NewStore()
	first := newTestProduct(t, "product-1", "Keyboard")
	created, err := store.Create(ctx, first, "request-1", "fingerprint-1")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	replay, err := store.Create(ctx, newTestProduct(t, "product-2", "Mouse"), "request-1", "fingerprint-1")
	if err != nil || replay.ID != created.ID {
		t.Fatalf("idempotent Create() = %#v, %v; want original product", replay, err)
	}
	if _, err := store.Create(ctx, newTestProduct(t, "product-3", "Desk"), "request-1", "different"); !errors.Is(err, catalogapp.ErrIdempotencyConflict) {
		t.Fatalf("Create() with changed fingerprint error = %v; want idempotency conflict", err)
	}
	if _, err := store.Create(ctx, first, "request-2", "fingerprint-2"); !errors.Is(err, ErrEntityAlreadyExists) {
		t.Fatalf("Create() with duplicate product ID error = %v; want duplicate entity", err)
	}
}

func TestStoreOrderIdempotencyDoesNotExposeStoredItems(t *testing.T) {
	ctx := context.Background()
	store := NewStore()
	order, err := orders.NewOrder("order-1", []orders.LineItem{{
		ProductID: "product-1", ProductName: "Keyboard", Quantity: 1,
		UnitPriceCents: 1200, Currency: "CAD",
	}}, time.Now())
	if err != nil {
		t.Fatalf("NewOrder() error = %v", err)
	}
	created, err := store.CreateOrder(ctx, order, "request-1", "fingerprint-1")
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}

	created.Items[0].ProductName = "mutated response"
	order.Items[0].ProductName = "mutated input"
	found, err := store.FindOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("FindOrder() error = %v", err)
	}
	if found.Items[0].ProductName != "Keyboard" {
		t.Fatalf("stored order item name = %q; want original snapshot", found.Items[0].ProductName)
	}

	replay, replayFound, err := store.FindOrderByIdempotency(ctx, "request-1", "fingerprint-1")
	if err != nil || !replayFound || replay.Items[0].ProductName != "Keyboard" {
		t.Fatalf("FindOrderByIdempotency() = %#v, %v, %v; want original order", replay, replayFound, err)
	}
	if _, err := store.CreateOrder(ctx, order, "request-1", "different"); !errors.Is(err, orderapp.ErrIdempotencyConflict) {
		t.Fatalf("CreateOrder() with changed fingerprint error = %v; want idempotency conflict", err)
	}
}

func TestStoreRejectsCanceledWrite(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	store := NewStore()
	if _, err := store.Create(ctx, newTestProduct(t, "product-1", "Keyboard"), "request-1", "fingerprint-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Create() with canceled context error = %v; want context.Canceled", err)
	}
	if _, err := store.Find(ctx, "product-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Find() with canceled context error = %v; want context.Canceled", err)
	}
}

func TestStoreConcurrentProductCreationReplaysIdempotently(t *testing.T) {
	const requests = 16
	store := NewStore()
	results := make(chan catalog.Product, requests)
	errorsFound := make(chan error, requests)
	var wait sync.WaitGroup
	products := make([]catalog.Product, requests)
	for index := range products {
		products[index] = newTestProduct(t, "product-"+strconv.Itoa(index), "Keyboard")
	}

	for index := range requests {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			created, err := store.Create(context.Background(), products[index], "request-1", "same-fingerprint")
			if err != nil {
				errorsFound <- err
				return
			}
			results <- created
		}(index)
	}
	wait.Wait()
	close(results)
	close(errorsFound)

	for err := range errorsFound {
		t.Errorf("concurrent Create() error = %v", err)
	}
	var originalID string
	for product := range results {
		if originalID == "" {
			originalID = product.ID
		} else if product.ID != originalID {
			t.Errorf("concurrent Create() returned product %q; want %q", product.ID, originalID)
		}
	}
}

func newTestProduct(t *testing.T, id, name string) catalog.Product {
	t.Helper()
	product, err := catalog.NewProduct(id, name, "", 100, "CAD", time.Now())
	if err != nil {
		t.Fatalf("NewProduct() error = %v", err)
	}
	return product
}
