package events

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/LeonardoCAlves/synechron/order-platform/internal/identity"
)

var ErrInvalidEnvelope = errors.New("invalid event envelope")

type Envelope struct {
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	Version     int             `json:"version"`
	AggregateID string          `json:"aggregateId"`
	OccurredAt  time.Time       `json:"occurredAt"`
	Data        json.RawMessage `json:"data"`
}

func New(eventType, aggregateID string, version int, occurredAt time.Time, data any) (Envelope, error) {
	payload, err := json.Marshal(data)
	if err != nil {
		return Envelope{}, err
	}
	id, err := identity.NewUUID()
	if err != nil {
		return Envelope{}, err
	}

	envelope := Envelope{
		ID:          id,
		Type:        eventType,
		Version:     version,
		AggregateID: aggregateID,
		OccurredAt:  occurredAt.UTC(),
		Data:        payload,
	}
	if err := envelope.Validate(); err != nil {
		return Envelope{}, err
	}
	return envelope, nil
}

func (envelope Envelope) Validate() error {
	if strings.TrimSpace(envelope.ID) == "" || strings.TrimSpace(envelope.Type) == "" ||
		strings.TrimSpace(envelope.AggregateID) == "" || envelope.Version < 1 ||
		envelope.OccurredAt.IsZero() || len(envelope.Data) == 0 || !json.Valid(envelope.Data) {
		return ErrInvalidEnvelope
	}
	return nil
}
