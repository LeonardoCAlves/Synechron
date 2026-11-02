package httpapi

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/LeonardoCAlves/synechron/order-platform/internal/catalog"
	"github.com/LeonardoCAlves/synechron/order-platform/internal/catalogapp"
)

type CatalogService interface {
	CreateProduct(context.Context, catalogapp.CreateProductInput, string) (catalog.Product, error)
	GetProduct(context.Context, string) (catalog.Product, error)
}

type ReadinessCheck func(context.Context) error

type createProductRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	PriceCents  int64  `json:"priceCents"`
	Currency    string `json:"currency"`
}

func NewCatalogHandler(service CatalogService, readiness ...ReadinessCheck) http.Handler {
	handler := &catalogHandler{service: service}
	if len(readiness) > 0 {
		handler.readiness = readiness[0]
	}
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		switch {
		case request.URL.Path == "/healthz":
			if request.Method != http.MethodGet {
				methodNotAllowed(w, http.MethodGet)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		case request.URL.Path == "/readyz":
			if request.Method != http.MethodGet {
				methodNotAllowed(w, http.MethodGet)
				return
			}
			handler.ready(w, request)
		case request.URL.Path == "/api/v1/products":
			if request.Method != http.MethodPost {
				methodNotAllowed(w, http.MethodPost)
				return
			}
			handler.create(w, request)
		case strings.HasPrefix(request.URL.Path, "/api/v1/products/"):
			if request.Method != http.MethodGet {
				methodNotAllowed(w, http.MethodGet)
				return
			}
			id := strings.TrimPrefix(request.URL.Path, "/api/v1/products/")
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

type catalogHandler struct {
	service   CatalogService
	readiness ReadinessCheck
}

func (handler *catalogHandler) ready(w http.ResponseWriter, request *http.Request) {
	if handler.readiness != nil {
		if err := handler.readiness(request.Context()); err != nil {
			log.Printf("catalog readiness check failed: %v", err)
			writeAPIError(w, http.StatusServiceUnavailable, "not_ready", "service dependencies are unavailable")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (handler *catalogHandler) create(w http.ResponseWriter, request *http.Request) {
	key, err := idempotencyKey(request)
	if writeDecodeError(w, err) {
		return
	}

	var body createProductRequest
	if err := decodeJSON(w, request, &body); err != nil {
		writeDecodeError(w, err)
		return
	}

	product, err := handler.service.CreateProduct(request.Context(), catalogapp.CreateProductInput{
		Name:        body.Name,
		Description: body.Description,
		PriceCents:  body.PriceCents,
		Currency:    body.Currency,
	}, key)
	if err != nil {
		writeCatalogError(w, err)
		return
	}

	w.Header().Set("Location", "/api/v1/products/"+product.ID)
	writeJSON(w, http.StatusCreated, product)
}

func (handler *catalogHandler) get(w http.ResponseWriter, request *http.Request, id string) {
	product, err := handler.service.GetProduct(request.Context(), id)
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, product)
}

func writeCatalogError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, catalog.ErrInvalidProduct):
		writeAPIError(w, http.StatusBadRequest, "invalid_product", "product fields are invalid")
	case errors.Is(err, catalogapp.ErrProductNotFound):
		writeAPIError(w, http.StatusNotFound, "product_not_found", "product was not found")
	case errors.Is(err, catalogapp.ErrInvalidIdempotencyKey):
		writeAPIError(w, http.StatusBadRequest, "invalid_idempotency_key", "a non-empty Idempotency-Key of at most 128 characters is required")
	case errors.Is(err, catalogapp.ErrIdempotencyConflict):
		writeAPIError(w, http.StatusConflict, "idempotency_conflict", "Idempotency-Key was already used with a different request")
	default:
		log.Printf("catalog request failed: %v", err)
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
	}
}
