package catalog

import (
	"errors"
	"testing"
	"time"
)

func TestNewProductTrimsAndNormalizesValues(t *testing.T) {
	now := time.Date(2026, time.October, 7, 18, 0, 0, 0, time.FixedZone("test", -3*60*60))

	product, err := NewProduct("product-1", "  Keyboard  ", "  Mechanical  ", 12999, " cad ", now)
	if err != nil {
		t.Fatalf("NewProduct() error = %v", err)
	}

	if product.Name != "Keyboard" || product.Description != "Mechanical" {
		t.Fatalf("product text was not trimmed: %#v", product)
	}
	if product.Currency != "CAD" || !product.CreatedAt.Equal(now.UTC()) {
		t.Fatalf("product normalization failed: %#v", product)
	}
	if !product.Active {
		t.Fatal("new product should be active")
	}
}

func TestNewProductRejectsInvalidValues(t *testing.T) {
	now := time.Date(2026, time.October, 7, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name        string
		id          string
		productName string
		price       int64
		currency    string
	}{
		{name: "missing id", productName: "Desk", price: 100, currency: "CAD"},
		{name: "blank name", id: "product-1", productName: "  ", price: 100, currency: "CAD"},
		{name: "negative price", id: "product-1", productName: "Desk", price: -1, currency: "CAD"},
		{name: "invalid currency", id: "product-1", productName: "Desk", price: 100, currency: "C$D"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewProduct(test.id, test.productName, "", test.price, test.currency, now)
			if !errors.Is(err, ErrInvalidProduct) {
				t.Fatalf("NewProduct() error = %v, want %v", err, ErrInvalidProduct)
			}
		})
	}
}
