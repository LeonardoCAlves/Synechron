package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/LeonardoCAlves/synechron/order-platform/internal/catalog"
	"github.com/LeonardoCAlves/synechron/order-platform/internal/catalogapp"
	"github.com/LeonardoCAlves/synechron/order-platform/internal/eventflow"
	"github.com/LeonardoCAlves/synechron/order-platform/internal/memory"
)

func TestCatalogHTTPCreateFetchAndIdempotency(t *testing.T) {
	handler := NewCatalogHandler(catalogapp.NewService(memory.NewStore()))
	body := `{"name":" Keyboard ","description":" Mechanical ","priceCents":12999,"currency":" cad "}`

	first := requestCatalog(handler, http.MethodPost, "/api/v1/products", body, "catalog-request-1")
	if first.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, want %d: %s", first.Code, http.StatusCreated, first.Body)
	}
	var created catalog.Product
	if err := json.Unmarshal(first.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created product: %v", err)
	}
	if created.Name != "Keyboard" || created.Currency != "CAD" || created.PriceCents != 12999 {
		t.Fatalf("created product was not normalized: %#v", created)
	}
	if got := first.Header().Get("Location"); got != "/api/v1/products/"+created.ID {
		t.Fatalf("Location = %q, want product path", got)
	}

	replay := requestCatalog(handler, http.MethodPost, "/api/v1/products", body, "catalog-request-1")
	if replay.Code != http.StatusCreated {
		t.Fatalf("replayed POST status = %d, want %d: %s", replay.Code, http.StatusCreated, replay.Body)
	}
	var replayed catalog.Product
	if err := json.Unmarshal(replay.Body.Bytes(), &replayed); err != nil {
		t.Fatalf("decode replayed product: %v", err)
	}
	if replayed.ID != created.ID {
		t.Fatalf("replayed product ID = %q, want original ID %q", replayed.ID, created.ID)
	}

	get := httptest.NewRecorder()
	handler.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/api/v1/products/"+created.ID, nil))
	if get.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d: %s", get.Code, http.StatusOK, get.Body)
	}

	conflict := requestCatalog(handler, http.MethodPost, "/api/v1/products",
		`{"name":"Keyboard","description":"Mechanical","priceCents":13000,"currency":"CAD"}`, "catalog-request-1")
	if conflict.Code != http.StatusConflict || !strings.Contains(conflict.Body.String(), `"idempotency_conflict"`) {
		t.Fatalf("changed request status/body = %d %s, want conflict", conflict.Code, conflict.Body)
	}
}

func TestCatalogHTTPRejectsInvalidBoundaryRequests(t *testing.T) {
	handler := NewCatalogHandler(catalogapp.NewService(memory.NewStore()))
	tests := []struct {
		name        string
		method      string
		path        string
		contentType string
		key         string
		body        string
		wantStatus  int
		wantCode    string
	}{
		{
			name: "missing idempotency key", method: http.MethodPost, path: "/api/v1/products",
			contentType: "application/json", body: `{"name":"Desk","currency":"CAD"}`,
			wantStatus: http.StatusBadRequest, wantCode: "invalid_idempotency_key",
		},
		{
			name: "unknown field", method: http.MethodPost, path: "/api/v1/products",
			contentType: "application/json", key: "request-2", body: `{"name":"Desk","currency":"CAD","extra":true}`,
			wantStatus: http.StatusBadRequest, wantCode: "invalid_request",
		},
		{
			name: "unsupported media type", method: http.MethodPost, path: "/api/v1/products",
			contentType: "text/plain", key: "request-3", body: `{"name":"Desk"}`,
			wantStatus: http.StatusUnsupportedMediaType, wantCode: "unsupported_media_type",
		},
		{
			name: "wrong method", method: http.MethodGet, path: "/api/v1/products",
			wantStatus: http.StatusMethodNotAllowed, wantCode: "method_not_allowed",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
			if test.contentType != "" {
				request.Header.Set("Content-Type", test.contentType)
			}
			if test.key != "" {
				request.Header.Set("Idempotency-Key", test.key)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus || !strings.Contains(response.Body.String(), `"`+test.wantCode+`"`) {
				t.Fatalf("status/body = %d %s, want %d with code %q", response.Code, response.Body, test.wantStatus, test.wantCode)
			}
			if response.Header().Get("Content-Type") != "application/json" {
				t.Fatalf("Content-Type = %q, want application/json", response.Header().Get("Content-Type"))
			}
		})
	}
}

func TestCatalogHTTPRejectsOversizedRequest(t *testing.T) {
	handler := NewCatalogHandler(catalogapp.NewService(memory.NewStore()))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/products", strings.NewReader(strings.Repeat("x", maxRequestBodyBytes+1)))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "oversized-request")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge || !strings.Contains(response.Body.String(), `"payload_too_large"`) {
		t.Fatalf("status/body = %d %s, want payload too large", response.Code, response.Body)
	}
}

func TestCatalogReadinessIsSeparateFromLiveness(t *testing.T) {
	handler := NewCatalogHandler(
		catalogapp.NewService(memory.NewStore()),
		func(context.Context) error { return errors.New("database unavailable") },
	)
	ready := httptest.NewRecorder()
	handler.ServeHTTP(ready, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if ready.Code != http.StatusServiceUnavailable || !strings.Contains(ready.Body.String(), `"not_ready"`) {
		t.Fatalf("readiness status/body = %d %s, want dependency failure", ready.Code, ready.Body)
	}

	live := httptest.NewRecorder()
	handler.ServeHTTP(live, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if live.Code != http.StatusOK {
		t.Fatalf("liveness status = %d, want %d", live.Code, http.StatusOK)
	}
}

func TestMetricsEndpointExposesPrometheusContentType(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", &eventflow.Metrics{})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	if response.Code != http.StatusOK || !strings.Contains(response.Header().Get("Content-Type"), "text/plain") ||
		!strings.Contains(response.Body.String(), "order_platform_outbox_oldest_pending_seconds") {
		t.Fatalf("metrics response = %d %#v %s", response.Code, response.Header(), response.Body)
	}
}

func requestCatalog(handler http.Handler, method, path, body, key string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", key)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
