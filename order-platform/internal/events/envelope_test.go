package events

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestEnvelopeValidate(t *testing.T) {
	envelope := Envelope{
		ID:          "event-1",
		Type:        "catalog.product.created.v1",
		Version:     1,
		AggregateID: "product-1",
		OccurredAt:  time.Now().UTC(),
		Data:        json.RawMessage(`{"name":"Keyboard"}`),
	}

	func TestNewBuildsVersionedEnvelopeWithUTCDate(t *testing.T) {
		occurredAt := time.Date(2026, time.October, 7, 18, 0, 0, 0, time.FixedZone("test", -3*60*60))
		envelope, err := New("catalog.product.created.v1", "product-1", 1, occurredAt, map[string]string{
			"name": "Keyboard",
		})
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		if err := envelope.Validate(); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
		if envelope.Version != 1 || !envelope.OccurredAt.Equal(occurredAt.UTC()) {
			t.Fatalf("event metadata was not normalized: %#v", envelope)
		}
	}
	if err := envelope.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	envelope.Data = json.RawMessage(`{"name":`)
	if err := envelope.Validate(); !errors.Is(err, ErrInvalidEnvelope) {
		t.Fatalf("Validate() error = %v, want %v", err, ErrInvalidEnvelope)
	}
}
