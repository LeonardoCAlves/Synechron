package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/LeonardoCAlves/synechron/order-platform/internal/catalog"
	"github.com/LeonardoCAlves/synechron/order-platform/internal/memory"
	"github.com/LeonardoCAlves/synechron/order-platform/internal/orderapp"
	"github.com/LeonardoCAlves/synechron/order-platform/internal/orders"
)

func TestOrderHTTPCreateSnapshotsProjectedProductAndIsIdempotent(t *testing.T) {
	store := memory.NewStore()
	product, err := catalog.NewProduct("product-1", "Keyboard", "Mechanical", 12999, "CAD", time.Now())
	if err != nil {
		t.Fatalf("NewProduct() error = %v", err)
	}
	if err := store.PutCatalogProduct(t.Context(), product); err != nil {
		t.Fatalf("PutCatalogProduct() error = %v", err)
	}
	handler := NewOrderHandler(orderapp.NewService(store, store))
	body := `{"items":[{"productId":"product-1","quantity":2}]}`

	first := requestOrder(handler, http.MethodPost, "/api/v1/orders", body, "order-request-1")
	if first.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, want %d: %s", first.Code, http.StatusCreated, first.Body)
	}
	var created orders.Order
	if err := json.Unmarshal(first.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created order: %v", err)
	}
	if created.TotalCents != 25998 || created.Items[0].ProductName != "Keyboard" || created.Items[0].UnitPriceCents != 12999 {
		t.Fatalf("order did not snapshot projected product: %#v", created)
	}

	if err := store.PutCatalogProduct(t.Context(), mustProduct(t, "product-1", "Updated keyboard", 14999)); err != nil {
		t.Fatalf("update projection: %v", err)
	}
	replay := requestOrder(handler, http.MethodPost, "/api/v1/orders", body, "order-request-1")
	if replay.Code != http.StatusCreated {
		t.Fatalf("replayed POST status = %d, want %d: %s", replay.Code, http.StatusCreated, replay.Body)
	}
	var replayed orders.Order
	if err := json.Unmarshal(replay.Body.Bytes(), &replayed); err != nil {
		t.Fatalf("decode replayed order: %v", err)
	}
	if replayed.ID != created.ID || replayed.Items[0].UnitPriceCents != 12999 {
		t.Fatalf("replay did not return original order snapshot: %#v", replayed)
	}

	get := httptest.NewRecorder()
	handler.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/api/v1/orders/"+created.ID, nil))
	if get.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d: %s", get.Code, http.StatusOK, get.Body)
	}

	conflict := requestOrder(handler, http.MethodPost, "/api/v1/orders",
		`{"items":[{"productId":"product-1","quantity":1}]}`, "order-request-1")
	if conflict.Code != http.StatusConflict || !strings.Contains(conflict.Body.String(), `"idempotency_conflict"`) {
		t.Fatalf("changed request status/body = %d %s, want conflict", conflict.Code, conflict.Body)
	}
}

func TestOrderHTTPReportsMissingProjectionAsRetryable(t *testing.T) {
	store := memory.NewStore()
	handler := NewOrderHandler(orderapp.NewService(store, store))
	response := requestOrder(handler, http.MethodPost, "/api/v1/orders",
		`{"items":[{"productId":"not-yet-projected","quantity":1}]}`, "order-request-2")

	if response.Code != http.StatusServiceUnavailable || response.Header().Get("Retry-After") != "1" ||
		!strings.Contains(response.Body.String(), `"catalog_projection_unavailable"`) {
		t.Fatalf("status/headers/body = %d %#v %s, want retryable projection error", response.Code, response.Header(), response.Body)
	}
}

func TestOrderHTTPRejectsInvalidRequestAndUnknownRoute(t *testing.T) {
	store := memory.NewStore()
	handler := NewOrderHandler(orderapp.NewService(store, store))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/orders", strings.NewReader(
		`{"items":[{"productId":"product-1","quantity":1,"unitPriceCents":1}]}`,
	))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "order-request-3")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"invalid_request"`) {
		t.Fatalf("status/body = %d %s, want invalid request", response.Code, response.Body)
	}

	notFound := httptest.NewRecorder()
	handler.ServeHTTP(notFound, httptest.NewRequest(http.MethodGet, "/unversioned/orders", nil))
	if notFound.Code != http.StatusNotFound || !strings.Contains(notFound.Body.String(), `"not_found"`) {
		t.Fatalf("status/body = %d %s, want not found", notFound.Code, notFound.Body)
	}
}

func requestOrder(handler http.Handler, method, path, body, key string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", key)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func mustProduct(t *testing.T, id, name string, priceCents int64) catalog.Product {
	t.Helper()
	product, err := catalog.NewProduct(id, name, "", priceCents, "CAD", time.Now())
	if err != nil {
		t.Fatalf("NewProduct() error = %v", err)
	}
	return product
}

func TestOrderHTTPResponseContainsStablePublicFields(t *testing.T) {
	store := memory.NewStore()
	if err := store.PutCatalogProduct(t.Context(), mustProduct(t, "product-1", "Desk", 100)); err != nil {
		t.Fatalf("PutCatalogProduct() error = %v", err)
	}
	handler := NewOrderHandler(orderapp.NewService(store, store))
	response := requestOrder(handler, http.MethodPost, "/api/v1/orders",
		`{"items":[{"productId":"product-1","quantity":1}]}`, "order-request-4")
	if response.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, want %d: %s", response.Code, http.StatusCreated, response.Body)
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode order response: %v", err)
	}
	for _, field := range []string{"id", "items", "totalCents", "currency", "status", "createdAt"} {
		if _, exists := payload[field]; !exists {
			t.Errorf("response missing field %q", field)
		}
	}
}
