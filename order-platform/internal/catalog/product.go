package catalog

import (
	"errors"
	"strings"
	"time"

	"github.com/LeonardoCAlves/synechron/order-platform/internal/money"
)

var ErrInvalidProduct = errors.New("invalid product")

type Product struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	PriceCents  int64     `json:"priceCents"`
	Currency    money.Currency `json:"currency"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"createdAt"`
}

func NewProduct(id, name, description string, priceCents int64, currency string, now time.Time) (Product, error) {
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	parsedCurrency, currencyErr := money.ParseCurrency(currency)

	if id == "" || name == "" || len(name) > 120 || len(description) > 2000 ||
		priceCents < 0 || currencyErr != nil || now.IsZero() {
		return Product{}, ErrInvalidProduct
	}

	return Product{
		ID:          id,
		Name:        name,
		Description: description,
		PriceCents:  priceCents,
		Currency:    parsedCurrency,
		Active:      true,
		CreatedAt:   now.UTC(),
	}, nil
}
