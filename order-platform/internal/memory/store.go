package memory

import (
	"context"
	"errors"
	"sync"

	"github.com/LeonardoCAlves/synechron/order-platform/internal/catalog"
	"github.com/LeonardoCAlves/synechron/order-platform/internal/catalogapp"
	"github.com/LeonardoCAlves/synechron/order-platform/internal/orderapp"
	"github.com/LeonardoCAlves/synechron/order-platform/internal/orders"
)

var ErrEntityAlreadyExists = errors.New("entity already exists")

type productIdempotencyRecord struct {
	fingerprint string
	product     catalog.Product
}

type orderIdempotencyRecord struct {
	fingerprint string
	order       orders.Order
}

type Store struct {
	mu                 sync.RWMutex
	products           map[string]catalog.Product
	orders             map[string]orders.Order
	productIdempotency map[string]productIdempotencyRecord
	orderIdempotency   map[string]orderIdempotencyRecord
}

func NewStore() *Store {
	return &Store{
		products:           make(map[string]catalog.Product),
		orders:             make(map[string]orders.Order),
		productIdempotency: make(map[string]productIdempotencyRecord),
		orderIdempotency:   make(map[string]orderIdempotencyRecord),
	}
}

func (store *Store) Find(ctx context.Context, id string) (catalog.Product, error) {
	if err := ctx.Err(); err != nil {
		return catalog.Product{}, err
	}
	store.mu.RLock()
	defer store.mu.RUnlock()
	product, exists := store.products[id]
	if !exists {
		return catalog.Product{}, catalogapp.ErrProductNotFound
	}
	return product, nil
}

func (store *Store) FindByIdempotency(ctx context.Context, key, fingerprint string) (catalog.Product, bool, error) {
	if err := ctx.Err(); err != nil {
		return catalog.Product{}, false, err
	}
	store.mu.RLock()
	defer store.mu.RUnlock()
	record, exists := store.productIdempotency[key]
	if !exists {
		return catalog.Product{}, false, nil
	}
	if record.fingerprint != fingerprint {
		return catalog.Product{}, false, catalogapp.ErrIdempotencyConflict
	}
	return record.product, true, nil
}

func (store *Store) Create(ctx context.Context, product catalog.Product, key, fingerprint string) (catalog.Product, error) {
	if err := ctx.Err(); err != nil {
		return catalog.Product{}, err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if record, exists := store.productIdempotency[key]; exists {
		if record.fingerprint != fingerprint {
			return catalog.Product{}, catalogapp.ErrIdempotencyConflict
		}
		return record.product, nil
	}
	if _, exists := store.products[product.ID]; exists {
		return catalog.Product{}, ErrEntityAlreadyExists
	}
	store.products[product.ID] = product
	store.productIdempotency[key] = productIdempotencyRecord{fingerprint: fingerprint, product: product}
	return product, nil
}

func (store *Store) FindCatalogProduct(ctx context.Context, id string) (catalog.Product, error) {
	if err := ctx.Err(); err != nil {
		return catalog.Product{}, err
	}
	store.mu.RLock()
	defer store.mu.RUnlock()
	product, exists := store.products[id]
	if !exists {
		return catalog.Product{}, orderapp.ErrCatalogProductNotFound
	}
	return product, nil
}

func (store *Store) PutCatalogProduct(ctx context.Context, product catalog.Product) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.products[product.ID] = product
	return nil
}

func (store *Store) FindOrder(ctx context.Context, id string) (orders.Order, error) {
	if err := ctx.Err(); err != nil {
		return orders.Order{}, err
	}
	store.mu.RLock()
	defer store.mu.RUnlock()
	order, exists := store.orders[id]
	if !exists {
		return orders.Order{}, orderapp.ErrOrderNotFound
	}
	return cloneOrder(order), nil
}

func (store *Store) FindOrderByIdempotency(ctx context.Context, key, fingerprint string) (orders.Order, bool, error) {
	if err := ctx.Err(); err != nil {
		return orders.Order{}, false, err
	}
	store.mu.RLock()
	defer store.mu.RUnlock()
	record, exists := store.orderIdempotency[key]
	if !exists {
		return orders.Order{}, false, nil
	}
	if record.fingerprint != fingerprint {
		return orders.Order{}, false, orderapp.ErrIdempotencyConflict
	}
	return cloneOrder(record.order), true, nil
}

func (store *Store) CreateOrder(ctx context.Context, order orders.Order, key, fingerprint string) (orders.Order, error) {
	if err := ctx.Err(); err != nil {
		return orders.Order{}, err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if record, exists := store.orderIdempotency[key]; exists {
		if record.fingerprint != fingerprint {
			return orders.Order{}, orderapp.ErrIdempotencyConflict
		}
		return cloneOrder(record.order), nil
	}
	if _, exists := store.orders[order.ID]; exists {
		return orders.Order{}, ErrEntityAlreadyExists
	}
	order = cloneOrder(order)
	store.orders[order.ID] = order
	store.orderIdempotency[key] = orderIdempotencyRecord{fingerprint: fingerprint, order: order}
	return cloneOrder(order), nil
}

func cloneOrder(order orders.Order) orders.Order {
	order.Items = append([]orders.LineItem(nil), order.Items...)
	return order
}

var (
	_ catalogapp.ProductRepository = (*Store)(nil)
	_ orderapp.OrderRepository     = (*Store)(nil)
	_ orderapp.CatalogProjection   = (*Store)(nil)
)
