package orders

import (
	"errors"
	"math"
	"testing"
	"time"
)

func TestNewOrderSnapshotsPricesAndCalculatesExactTotal(t *testing.T) {
	now := time.Date(2026, time.October, 7, 18, 0, 0, 0, time.FixedZone("test", -3*60*60))
	items := []LineItem{
		{ProductID: "p-1", ProductName: "Keyboard", Quantity: 2, UnitPriceCents: 12999, Currency: "cad"},
		{ProductID: "p-2", ProductName: "Mouse", Quantity: 1, UnitPriceCents: 5999, Currency: "CAD"},
	}

	order, err := NewOrder("order-1", items, now)
	if err != nil {
		t.Fatalf("NewOrder() error = %v", err)
	}
	if order.TotalCents != 31997 || order.Currency != "CAD" || order.Status != StatusPending {
		t.Fatalf("unexpected order values: %#v", order)
	}
	if !order.CreatedAt.Equal(now.UTC()) || order.Items[0].ProductName != "Keyboard" {
		t.Fatalf("order snapshot was not preserved: %#v", order)
	}
}

func TestNewOrderRejectsInvalidAndAmbiguousItems(t *testing.T) {
	now := time.Date(2026, time.October, 7, 0, 0, 0, 0, time.UTC)
	valid := LineItem{ProductID: "p-1", ProductName: "Keyboard", Quantity: 1, UnitPriceCents: 100, Currency: "CAD"}
	tests := []struct {
		name  string
		items []LineItem
	}{
		{name: "empty", items: nil},
		{name: "non-positive quantity", items: []LineItem{{ProductID: "p-1", ProductName: "Keyboard", Quantity: 0, Currency: "CAD"}}},
		{name: "negative price", items: []LineItem{{ProductID: "p-1", ProductName: "Keyboard", Quantity: 1, UnitPriceCents: -1, Currency: "CAD"}}},
		{name: "mixed currencies", items: []LineItem{valid, {ProductID: "p-2", ProductName: "Mouse", Quantity: 1, Currency: "USD"}}},
		{name: "duplicate product", items: []LineItem{valid, valid}},
		{name: "quantity overflow", items: []LineItem{{ProductID: "p-1", ProductName: "Keyboard", Quantity: maxQuantity + 1, Currency: "CAD"}}},
		{name: "line total overflow", items: []LineItem{{ProductID: "p-1", ProductName: "Keyboard", Quantity: 2, UnitPriceCents: math.MaxInt64, Currency: "CAD"}}},
		{name: "aggregate total overflow", items: []LineItem{
			{ProductID: "p-1", ProductName: "Keyboard", Quantity: 1, UnitPriceCents: math.MaxInt64, Currency: "CAD"},
			{ProductID: "p-2", ProductName: "Mouse", Quantity: 1, UnitPriceCents: 1, Currency: "CAD"},
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewOrder("order-1", test.items, now)
			if !errors.Is(err, ErrInvalidOrder) {
				t.Fatalf("NewOrder() error = %v, want %v", err, ErrInvalidOrder)
			}
		})
	}
}

func TestOrderStateTransitionsAreExplicitAndTerminal(t *testing.T) {
	order, err := NewOrder("order-1", []LineItem{
		{ProductID: "p-1", ProductName: "Keyboard", Quantity: 1, UnitPriceCents: 100, Currency: "CAD"},
	}, time.Now())
	if err != nil {
		t.Fatalf("NewOrder() error = %v", err)
	}
	if err := order.TransitionTo(StatusAccepted); err != nil {
		t.Fatalf("TransitionTo(accepted) error = %v", err)
	}
	if err := order.TransitionTo(StatusRejected); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("terminal transition error = %v, want %v", err, ErrInvalidTransition)
	}
}
