package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"
	"strings"
)

const maxRequestBodyBytes = 1 << 20

var (
	errInvalidRequest     = errors.New("invalid request")
	errUnsupportedMedia   = errors.New("unsupported media type")
	errPayloadTooLarge    = errors.New("request body too large")
	errInvalidIdempotency = errors.New("invalid idempotency key")
)

type errorResponse struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func decodeJSON(w http.ResponseWriter, request *http.Request, destination any) error {
	contentType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		return errUnsupportedMedia
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, request.Body, maxRequestBodyBytes))
	if err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			return errPayloadTooLarge
		}
		return errInvalidRequest
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			return errPayloadTooLarge
		}
		return errInvalidRequest
	}

	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			return errPayloadTooLarge
		}
		return errInvalidRequest
	}
	return nil
}

func idempotencyKey(request *http.Request) (string, error) {
	key := strings.TrimSpace(request.Header.Get("Idempotency-Key"))
	if key == "" || len(key) > 128 {
		return "", errInvalidIdempotency
	}
	return key, nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode HTTP response: %v", err)
	}
}

func writeAPIError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorResponse{Error: apiError{Code: code, Message: message}})
}

func methodNotAllowed(w http.ResponseWriter, method string) {
	w.Header().Set("Allow", method)
	writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method is not allowed for this endpoint")
}

func notFound(w http.ResponseWriter) {
	writeAPIError(w, http.StatusNotFound, "not_found", "resource not found")
}

func writeDecodeError(w http.ResponseWriter, err error) bool {
	switch {
	case errors.Is(err, errUnsupportedMedia):
		writeAPIError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "content type must be application/json")
	case errors.Is(err, errPayloadTooLarge):
		writeAPIError(w, http.StatusRequestEntityTooLarge, "payload_too_large", "request body exceeds the 1 MiB limit")
	case errors.Is(err, errInvalidIdempotency):
		writeAPIError(w, http.StatusBadRequest, "invalid_idempotency_key", "a non-empty Idempotency-Key of at most 128 characters is required")
	case errors.Is(err, errInvalidRequest):
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "request body must contain one valid JSON object with supported fields")
	default:
		return false
	}
	return true
}
