package catalogapp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/LeonardoCAlves/synechron/order-platform/internal/catalog"
	"github.com/LeonardoCAlves/synechron/order-platform/internal/identity"
	"github.com/LeonardoCAlves/synechron/order-platform/internal/money"
)

var (
	ErrProductNotFound       = errors.New("product not found")
	ErrInvalidIdempotencyKey = errors.New("invalid idempotency key")
	ErrIdempotencyConflict   = errors.New("idempotency key was used with a different request")
)

const maxIdempotencyKeyLength = 128

type ProductRepository interface {
	Find(context.Context, string) (catalog.Product, error)
	FindByIdempotency(context.Context, string, string) (catalog.Product, bool, error)
	Create(context.Context, catalog.Product, string, string) (catalog.Product, error)
}

type CreateProductInput struct {
	Name        string
	Description string
	PriceCents  int64
	Currency    string
}

type Service struct {
	products ProductRepository
}

func NewService(products ProductRepository) *Service {
	return &Service{products: products}
}

func (service *Service) CreateProduct(ctx context.Context, input CreateProductInput, key string) (catalog.Product, error) {
	key = strings.TrimSpace(key)
	if key == "" || len(key) > maxIdempotencyKeyLength {
		return catalog.Product{}, ErrInvalidIdempotencyKey
	}

	normalized, err := catalog.NewProduct("candidate", input.Name, input.Description, input.PriceCents, input.Currency, time.Unix(1, 0))
	if err != nil {
		return catalog.Product{}, err
	}
	fingerprint, err := fingerprint(struct {
		Name        string
		Description string
		PriceCents  int64
		Currency    money.Currency
	}{
		Name:        normalized.Name,
		Description: normalized.Description,
		PriceCents:  normalized.PriceCents,
		Currency:    normalized.Currency,
	})
	if err != nil {
		return catalog.Product{}, err
	}

	if product, found, err := service.products.FindByIdempotency(ctx, key, fingerprint); err != nil || found {
		return product, err
	}

	id, err := identity.NewUUID()
	if err != nil {
		return catalog.Product{}, err
	}
	product, err := catalog.NewProduct(id, input.Name, input.Description, input.PriceCents, input.Currency, time.Now())
	if err != nil {
		return catalog.Product{}, err
	}
	return service.products.Create(ctx, product, key, fingerprint)
}

func (service *Service) GetProduct(ctx context.Context, id string) (catalog.Product, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return catalog.Product{}, catalog.ErrInvalidProduct
	}
	return service.products.Find(ctx, id)
}

func fingerprint(value any) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:]), nil
}
