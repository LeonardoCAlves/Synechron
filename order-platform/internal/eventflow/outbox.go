package eventflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
	"time"

	"github.com/LeonardoCAlves/synechron/order-platform/internal/events"
	"github.com/LeonardoCAlves/synechron/order-platform/internal/postgres"
	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	outboxBatchSize   = 50
	maxOutboxAttempts = 12
	maxOutboxLease    = 30 * time.Second
)

type pendingOutbox interface {
	ClaimPending(context.Context, int, time.Duration) ([]postgres.OutboxMessage, error)
	MarkPublished(context.Context, string) error
	RecordFailure(context.Context, string, string, time.Duration, int) (bool, error)
	Stats(context.Context) (postgres.OutboxStats, error)
}

type Publisher struct {
	client  *kgo.Client
	outbox  pendingOutbox
	topic   string
	metrics *Metrics
}

func NewPublisher(brokers []string, topic string, outbox pendingOutbox, metrics *Metrics) (*Publisher, error) {
	topic = strings.TrimSpace(topic)
	client, err := newKafkaClient(brokers, kgo.RequiredAcks(kgo.AllISRAcks()))
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(topic) == "" {
		client.Close()
		return nil, errors.New("outbox Kafka topic is required")
	}
	return &Publisher{client: client, outbox: outbox, topic: topic, metrics: metrics}, nil
}

func (publisher *Publisher) Ping(ctx context.Context) error {
	if err := publisher.client.Ping(ctx); err != nil {
		return fmt.Errorf("ping Kafka producer: %w", err)
	}
	return nil
}

func (publisher *Publisher) Close() {
	publisher.client.Close()
}

func (publisher *Publisher) Run(ctx context.Context) error {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	statsTicker := time.NewTicker(15 * time.Second)
	defer statsTicker.Stop()

	for {
		if err := publisher.publishBatch(ctx); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("outbox polling failed", "topic", publisher.topic, "error", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		case <-statsTicker.C:
			if publisher.metrics == nil {
				continue
			}
			stats, err := publisher.outbox.Stats(ctx)
			if err != nil {
				slog.Error("outbox metrics query failed", "topic", publisher.topic, "error", err)
				continue
			}
			publisher.metrics.setOutboxStats(
				stats.Pending,
				int64(stats.OldestPendingAge.Seconds()),
				stats.DeadLettered,
			)
		}
	}
}

func (publisher *Publisher) publishBatch(ctx context.Context) error {
	messages, err := publisher.outbox.ClaimPending(ctx, outboxBatchSize, maxOutboxLease)
	if err != nil {
		return err
	}
	for _, message := range messages {
		if err := ctx.Err(); err != nil {
			return err
		}
		var envelope events.Envelope
		err := json.Unmarshal(message.Payload, &envelope)
		if err == nil {
			err = envelope.Validate()
		}
		if err == nil && (envelope.ID != message.ID || envelope.Type != message.EventType ||
			envelope.AggregateID != message.AggregateID ||
			envelope.AggregateVersion != message.AggregateVersion) {
			err = errors.New("outbox metadata does not match event envelope")
		}
		if err != nil {
			publisher.recordFailure(ctx, message, fmt.Errorf("validate outbox event: %w", err), 1)
			continue
		}

		record := &kgo.Record{
			Topic: publisher.topic,
			Key:   []byte(message.AggregateID),
			Value: message.Payload,
		}
		if err := publisher.client.ProduceSync(ctx, record).FirstErr(); err != nil {
			publisher.recordFailure(ctx, message, err, maxOutboxAttempts)
			continue
		}
		if err := publisher.outbox.MarkPublished(ctx, message.ID); err != nil {
			slog.Error("Kafka event was published but outbox acknowledgement failed",
				"event_id", message.ID, "topic", publisher.topic, "error", err)
			continue
		}
		if publisher.metrics != nil {
			publisher.metrics.published.Add(1)
		}
	}
	return nil
}

func (publisher *Publisher) recordFailure(ctx context.Context, message postgres.OutboxMessage, cause error, maxAttempts int) {
	nextAttempt := retryDelay(message.Attempts)
	if maxAttempts == 1 || message.Attempts+1 >= maxAttempts {
		nextAttempt = 0
	}
	deadLettered, err := publisher.outbox.RecordFailure(ctx, message.ID, cause.Error(), nextAttempt, maxAttempts)
	if err != nil {
		slog.Error("could not update failed outbox event", "event_id", message.ID, "error", err)
		return
	}
	if publisher.metrics != nil {
		publisher.metrics.retried.Add(1)
		if deadLettered {
			publisher.metrics.deadLettered.Add(1)
		}
	}
	if deadLettered {
		slog.Error("outbox event moved to manual recovery",
			"event_id", message.ID, "event_type", message.EventType, "error", cause)
		return
	}
	slog.Warn("outbox event delivery will be retried",
		"event_id", message.ID, "event_type", message.EventType, "error", cause, "retry_after", nextAttempt)
}

func retryDelay(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}
	exponent := attempt
	if exponent > 8 {
		exponent = 8
	}
	base := time.Second * time.Duration(1<<exponent)
	jitter := time.Duration(rand.Int63n(int64(base/2) + 1))
	delay := base/2 + jitter
	if delay > 5*time.Minute {
		return 5 * time.Minute
	}
	return delay
}
