package orderapp

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
	"github.com/LeonardoCAlves/synechron/order-platform/internal/orders"
)

var (
	ErrOrderNotFound          = errors.New("order not found")
	ErrCatalogProductNotFound = errors.New("catalog product is not in the local projection")
	ErrInactiveProduct        = errors.New("catalog product is inactive")
	ErrInvalidIdempotencyKey  = errors.New("invalid idempotency key")
	ErrIdempotencyConflict    = errors.New("idempotency key was used with a different request")
)

const maxIdempotencyKeyLength = 128

type OrderRepository interface {
	FindOrder(context.Context, string) (orders.Order, error)
	FindOrderByIdempotency(context.Context, string, string) (orders.Order, bool, error)
	CreateOrder(context.Context, orders.Order, string, string) (orders.Order, error)
}

type CatalogProjection interface {
	FindCatalogProduct(context.Context, string) (catalog.Product, error)
}

type CreateOrderItemInput struct {
	ProductID string `json:"productId"`
	Quantity  int64  `json:"quantity"`
}

type CreateOrderInput struct {
	Items []CreateOrderItemInput `json:"items"`
}

type Service struct {
	orders  OrderRepository
	catalog CatalogProjection
}

func NewService(orders OrderRepository, catalogProjection CatalogProjection) *Service {
	return &Service{orders: orders, catalog: catalogProjection}
}

func (service *Service) CreateOrder(ctx context.Context, input CreateOrderInput, key string) (orders.Order, error) {
	key = strings.TrimSpace(key)
	if key == "" || len(key) > maxIdempotencyKeyLength {
		return orders.Order{}, ErrInvalidIdempotencyKey
	}
	if len(input.Items) == 0 || len(input.Items) > orders.MaxOrderLines {
		return orders.Order{}, orders.ErrInvalidOrder
	}

	normalizedItems := make([]CreateOrderItemInput, len(input.Items))
	seenProducts := make(map[string]struct{}, len(input.Items))
	for index, item := range input.Items {
		item.ProductID = strings.TrimSpace(item.ProductID)
		if item.ProductID == "" || item.Quantity < 1 || item.Quantity > orders.MaxQuantity {
			return orders.Order{}, orders.ErrInvalidOrder
		}
		if _, exists := seenProducts[item.ProductID]; exists {
			return orders.Order{}, orders.ErrInvalidOrder
		}
		seenProducts[item.ProductID] = struct{}{}
		normalizedItems[index] = item
	}

	fingerprint, err := orderFingerprint(normalizedItems)
	if err != nil {
		return orders.Order{}, err
	}
	if order, found, err := service.orders.FindOrderByIdempotency(ctx, key, fingerprint); err != nil || found {
		return order, err
	}

	lineItems := make([]orders.LineItem, len(normalizedItems))
	for index, item := range normalizedItems {
		product, err := service.catalog.FindCatalogProduct(ctx, item.ProductID)
		if err != nil {
			return orders.Order{}, err
		}
		if !product.Active {
			return orders.Order{}, ErrInactiveProduct
		}
		lineItems[index] = orders.LineItem{
			ProductID:      product.ID,
			ProductName:    product.Name,
			Quantity:       item.Quantity,
			UnitPriceCents: product.PriceCents,
			Currency:       money.Currency(product.Currency),
		}
	}

	id, err := identity.NewUUID()
	if err != nil {
		return orders.Order{}, err
	}
	order, err := orders.NewOrder(id, lineItems, time.Now())
	if err != nil {
		return orders.Order{}, err
	}
	return service.orders.CreateOrder(ctx, order, key, fingerprint)
}

func (service *Service) GetOrder(ctx context.Context, id string) (orders.Order, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return orders.Order{}, orders.ErrInvalidOrder
	}
	return service.orders.FindOrder(ctx, id)
}

func orderFingerprint(items []CreateOrderItemInput) (string, error) {
	payload, err := json.Marshal(items)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:]), nil
}
