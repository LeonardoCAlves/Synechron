package eventflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/LeonardoCAlves/synechron/order-platform/internal/events"
	"github.com/LeonardoCAlves/synechron/order-platform/internal/postgres"
	"github.com/twmb/franz-go/pkg/kgo"
)

const catalogConsumerGroup = "order-service.catalog-projection.v1"

type catalogEventApplier interface {
	ApplyCatalogEvent(context.Context, events.Envelope) (bool, error)
}

type Consumer struct {
	client     *kgo.Client
	deadLetter *kgo.Client
	applier    catalogEventApplier
	topic      string
	deadTopic  string
	metrics    *Metrics
}

func NewCatalogConsumer(brokers []string, topic string, applier catalogEventApplier, metrics *Metrics) (*Consumer, error) {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return nil, errors.New("catalog Kafka topic is required")
	}
	client, err := newKafkaClient(
		brokers,
		kgo.ConsumerGroup(catalogConsumerGroup),
		kgo.ConsumeTopics(topic),
		kgo.DisableAutoCommit(),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
	)
	if err != nil {
		return nil, err
	}
	deadLetter, err := newKafkaClient(brokers, kgo.RequiredAcks(kgo.AllISRAcks()))
	if err != nil {
		client.Close()
		return nil, err
	}
	return &Consumer{
		client: client, deadLetter: deadLetter, applier: applier,
		topic: topic, deadTopic: topic + ".dead-letter", metrics: metrics,
	}, nil
}

func (consumer *Consumer) Ping(ctx context.Context) error {
	if err := consumer.client.Ping(ctx); err != nil {
		return fmt.Errorf("ping Kafka consumer: %w", err)
	}
	return nil
}

func (consumer *Consumer) Close() {
	consumer.client.Close()
	consumer.deadLetter.Close()
}

func (consumer *Consumer) Run(ctx context.Context) error {
	for {
		fetches := consumer.client.PollFetches(ctx)
		if ctx.Err() != nil {
			return nil
		}
		fetches.EachError(func(topic string, partition int32, err error) {
			slog.Error("Kafka catalog fetch failed", "topic", topic, "partition", partition, "error", err)
		})
		if err := consumer.processFetches(ctx, fetches); err != nil {
			slog.Error("catalog event processing failed; offset remains uncommitted",
				"topic", consumer.topic, "error", err)
			if !waitContext(ctx, time.Second) {
				return nil
			}
		}
	}
}

func (consumer *Consumer) processFetches(ctx context.Context, fetches kgo.Fetches) error {
	records := fetches.Records()
	sort.SliceStable(records, func(i, j int) bool {
		if records[i].Partition != records[j].Partition {
			return records[i].Partition < records[j].Partition
		}
		return records[i].Offset < records[j].Offset
	})
	for _, record := range records {
		if err := consumer.processRecord(ctx, record); err != nil {
			return fmt.Errorf("process partition %d offset %d: %w", record.Partition, record.Offset, err)
		}
	}
	return nil
}

func (consumer *Consumer) processRecord(ctx context.Context, record *kgo.Record) error {
	if len(record.Value) > 1<<20 {
		return consumer.deadLetterRecord(ctx, record, errors.New("event exceeds the 1 MiB processing limit"))
	}
	var envelope events.Envelope
	if err := json.Unmarshal(record.Value, &envelope); err != nil {
		return consumer.deadLetterRecord(ctx, record, fmt.Errorf("decode event envelope: %w", err))
	}
	if err := envelope.Validate(); err != nil {
		return consumer.deadLetterRecord(ctx, record, err)
	}
	applied, err := consumer.applier.ApplyCatalogEvent(ctx, envelope)
	if errors.Is(err, postgres.ErrPoisonCatalogEvent) {
		return consumer.deadLetterRecord(ctx, record, err)
	}
	if err != nil {
		return err
	}
	if err := consumer.client.CommitRecords(ctx, record); err != nil {
		return fmt.Errorf("commit Kafka offset after database commit: %w", err)
	}
	if consumer.metrics != nil {
		if applied {
			consumer.metrics.consumed.Add(1)
		} else {
			consumer.metrics.duplicates.Add(1)
		}
	}
	return nil
}

func (consumer *Consumer) deadLetterRecord(ctx context.Context, source *kgo.Record, cause error) error {
	deadLetter := &kgo.Record{
		Topic: consumer.deadTopic,
		Key:   append([]byte(nil), source.Key...),
		Value: append([]byte(nil), source.Value...),
		Headers: []kgo.RecordHeader{
			{Key: "source-topic", Value: []byte(source.Topic)},
			{Key: "source-partition", Value: []byte(fmt.Sprint(source.Partition))},
			{Key: "source-offset", Value: []byte(fmt.Sprint(source.Offset))},
			{Key: "failure-reason", Value: []byte(cause.Error())},
		},
	}
	if err := consumer.deadLetter.ProduceSync(ctx, deadLetter).FirstErr(); err != nil {
		return fmt.Errorf("publish poison event to dead-letter topic: %w", err)
	}
	if err := consumer.client.CommitRecords(ctx, source); err != nil {
		return fmt.Errorf("commit offset after dead-letter publication: %w", err)
	}
	if consumer.metrics != nil {
		consumer.metrics.poisoned.Add(1)
	}
	slog.Error("catalog event moved to Kafka dead-letter topic",
		"source_topic", source.Topic, "source_partition", source.Partition, "source_offset", source.Offset,
		"dead_letter_topic", consumer.deadTopic, "reason", cause)
	return nil
}

func waitContext(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
