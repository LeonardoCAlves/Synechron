package httpapi

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/LeonardoCAlves/synechron/order-platform/internal/orderapp"
	"github.com/LeonardoCAlves/synechron/order-platform/internal/orders"
)

type OrderService interface {
	CreateOrder(context.Context, orderapp.CreateOrderInput, string) (orders.Order, error)
	GetOrder(context.Context, string) (orders.Order, error)
}

func NewOrderHandler(service OrderService) http.Handler {
	handler := &orderHandler{service: service}
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		switch {
		case request.URL.Path == "/healthz":
			if request.Method != http.MethodGet {
				methodNotAllowed(w, http.MethodGet)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		case request.URL.Path == "/api/v1/orders":
			if request.Method != http.MethodPost {
				methodNotAllowed(w, http.MethodPost)
				return
			}
			handler.create(w, request)
		case strings.HasPrefix(request.URL.Path, "/api/v1/orders/"):
			if request.Method != http.MethodGet {
				methodNotAllowed(w, http.MethodGet)
				return
			}
			id := strings.TrimPrefix(request.URL.Path, "/api/v1/orders/")
			if id == "" || strings.Contains(id, "/") {
				notFound(w)
				return
			}
			handler.get(w, request, id)
		default:
			notFound(w)
		}
	})
}

type orderHandler struct {
	service OrderService
}

func (handler *orderHandler) create(w http.ResponseWriter, request *http.Request) {
	key, err := idempotencyKey(request)
	if writeDecodeError(w, err) {
		return
	}

	var body orderapp.CreateOrderInput
	if err := decodeJSON(w, request, &body); err != nil {
		writeDecodeError(w, err)
		return
	}
	order, err := handler.service.CreateOrder(request.Context(), body, key)
	if err != nil {
		writeOrderError(w, err)
		return
	}

	w.Header().Set("Location", "/api/v1/orders/"+order.ID)
	writeJSON(w, http.StatusCreated, order)
}

func (handler *orderHandler) get(w http.ResponseWriter, request *http.Request, id string) {
	order, err := handler.service.GetOrder(request.Context(), id)
	if err != nil {
		writeOrderError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func writeOrderError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, orders.ErrInvalidOrder):
		writeAPIError(w, http.StatusBadRequest, "invalid_order", "order fields are invalid")
	case errors.Is(err, orderapp.ErrOrderNotFound):
		writeAPIError(w, http.StatusNotFound, "order_not_found", "order was not found")
	case errors.Is(err, orderapp.ErrCatalogProductNotFound):
		w.Header().Set("Retry-After", "1")
		writeAPIError(w, http.StatusServiceUnavailable, "catalog_projection_unavailable", "product is not available in the local catalog projection yet")
	case errors.Is(err, orderapp.ErrInactiveProduct):
		writeAPIError(w, http.StatusConflict, "product_inactive", "inactive products cannot be ordered")
	case errors.Is(err, orderapp.ErrInvalidIdempotencyKey):
		writeAPIError(w, http.StatusBadRequest, "invalid_idempotency_key", "a non-empty Idempotency-Key of at most 128 characters is required")
	case errors.Is(err, orderapp.ErrIdempotencyConflict):
		writeAPIError(w, http.StatusConflict, "idempotency_conflict", "Idempotency-Key was already used with a different request")
	default:
		log.Printf("order request failed: %v", err)
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
	}
}
