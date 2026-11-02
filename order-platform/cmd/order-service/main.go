package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/LeonardoCAlves/synechron/order-platform/internal/eventflow"
	"github.com/LeonardoCAlves/synechron/order-platform/internal/httpapi"
	"github.com/LeonardoCAlves/synechron/order-platform/internal/httpserver"
	"github.com/LeonardoCAlves/synechron/order-platform/internal/orderapp"
	"github.com/LeonardoCAlves/synechron/order-platform/internal/postgres"
)

func main() {
	address := os.Getenv("HTTP_ADDR")
	if address == "" {
		address = "127.0.0.1:8082"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	startupContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	db, err := postgres.Open(startupContext, databaseURL)
	if err != nil {
		cancel()
		log.Fatal(err)
	}
	defer db.Close()
	if err := postgres.MigrateOrders(startupContext, db); err != nil {
		cancel()
		log.Fatal(err)
	}
	cancel()

	brokers := strings.Split(envOrDefault("KAFKA_BROKERS", "localhost:9092"), ",")
	catalogTopic := envOrDefault("CATALOG_EVENTS_TOPIC", "catalog.product-events.v1")
	orderTopic := envOrDefault("ORDER_EVENTS_TOPIC", "order.events.v1")
	metrics := &eventflow.Metrics{}
	store := postgres.NewOrderStore(db)
	consumer, err := eventflow.NewCatalogConsumer(brokers, catalogTopic, store, metrics)
	if err != nil {
		log.Fatal(err)
	}
	defer consumer.Close()
	publisher, err := eventflow.NewPublisher(brokers, orderTopic, postgres.NewOutboxStore(db), metrics)
	if err != nil {
		log.Fatal(err)
	}
	defer publisher.Close()

	service := orderapp.NewService(store, store)
	api := httpapi.NewOrderHandler(service, func(ctx context.Context) error {
		if err := db.PingContext(ctx); err != nil {
			return err
		}
		pingContext, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		if err := consumer.Ping(pingContext); err != nil {
			return err
		}
		return publisher.Ping(pingContext)
	})
	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics)
	mux.Handle("/", api)
	log.Printf("order-service listening on %s", address)
	if err := httpserver.RunWithWorkers(address, mux, publisher.Run, consumer.Run); err != nil {
		log.Fatal(err)
	}
}

func envOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
