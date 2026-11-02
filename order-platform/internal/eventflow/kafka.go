package eventflow

import (
	"errors"
	"fmt"
	"strings"

	"github.com/twmb/franz-go/pkg/kgo"
)

func newKafkaClient(brokers []string, options ...kgo.Opt) (*kgo.Client, error) {
	normalized := make([]string, 0, len(brokers))
	for _, broker := range brokers {
		broker = strings.TrimSpace(broker)
		if broker != "" {
			normalized = append(normalized, broker)
		}
	}
	if len(normalized) == 0 {
		return nil, errors.New("at least one Kafka broker is required")
	}
	client, err := kgo.NewClient(append([]kgo.Opt{kgo.SeedBrokers(normalized...)}, options...)...)
	if err != nil {
		return nil, fmt.Errorf("create Kafka client: %w", err)
	}
	return client, nil
}
