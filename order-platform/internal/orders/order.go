package orders

import (
	"errors"
	"strings"
	"time"

	"github.com/LeonardoCAlves/synechron/order-platform/internal/money"
)

var (
	ErrInvalidOrder     = errors.New("invalid order")
	ErrInvalidTransition = errors.New("invalid order status transition")
)

const (
	StatusPending  = "pending"
	StatusAccepted = "accepted"
	StatusRejected = "rejected"

	maxOrderLines = 100
	maxQuantity   = 1000
)

type LineItem struct {
	ProductID       string `json:"productId"`
	ProductName     string `json:"productName"`
	Quantity        int64  `json:"quantity"`
	UnitPriceCents  int64  `json:"unitPriceCents"`
	Currency        money.Currency `json:"currency"`
}

type Order struct {
	ID         string     `json:"id"`
	Items      []LineItem `json:"items"`
	TotalCents int64      `json:"totalCents"`
	Currency   money.Currency `json:"currency"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"createdAt"`
}

func NewOrder(id string, items []LineItem, now time.Time) (Order, error) {
	id = strings.TrimSpace(id)
	if id == "" || len(items) == 0 || len(items) > maxOrderLines || now.IsZero() {
		return Order{}, ErrInvalidOrder
	}

	currency, err := money.ParseCurrency(string(items[0].Currency))
	if err != nil {
		return Order{}, ErrInvalidOrder
	}

	total := int64(0)
	seenProducts := make(map[string]struct{}, len(items))
	snapshots := make([]LineItem, len(items))
	for index, item := range items {
		item.ProductID = strings.TrimSpace(item.ProductID)
		item.ProductName = strings.TrimSpace(item.ProductName)
		itemCurrency, err := money.ParseCurrency(string(item.Currency))
		if item.ProductID == "" || item.ProductName == "" || item.Quantity < 1 || item.Quantity > maxQuantity ||
			item.UnitPriceCents < 0 || err != nil || itemCurrency != currency {
			return Order{}, ErrInvalidOrder
		}
		if _, exists := seenProducts[item.ProductID]; exists {
			return Order{}, ErrInvalidOrder
		}
		seenProducts[item.ProductID] = struct{}{}

		lineTotal, err := money.MultiplyMinorUnits(item.UnitPriceCents, item.Quantity)
		if err != nil {
			return Order{}, ErrInvalidOrder
		}
		total, err = money.AddMinorUnits(total, lineTotal)
		if err != nil {
			return Order{}, ErrInvalidOrder
		}
		item.Currency = itemCurrency
		snapshots[index] = item
	}

	return Order{
		ID:         id,
		Items:      snapshots,
		TotalCents: total,
		Currency:   currency,
		Status:     StatusPending,
		CreatedAt:  now.UTC(),
	}, nil
}

func (order *Order) TransitionTo(status string) error {
	if order == nil || order.Status != StatusPending || (status != StatusAccepted && status != StatusRejected) {
		return ErrInvalidTransition
	}
	order.Status = status
	return nil
}
