package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/LeonardoCAlves/synechron/order-platform/internal/catalogapp"
	"github.com/LeonardoCAlves/synechron/order-platform/internal/eventflow"
	"github.com/LeonardoCAlves/synechron/order-platform/internal/httpapi"
	"github.com/LeonardoCAlves/synechron/order-platform/internal/httpserver"
	"github.com/LeonardoCAlves/synechron/order-platform/internal/postgres"
)

func main() {
	address := os.Getenv("HTTP_ADDR")
	if address == "" {
		address = "127.0.0.1:8081"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	startupContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	db, err := postgres.Open(startupContext, databaseURL)
	if err != nil {
		cancel()
		log.Fatal(err)
	}
	defer db.Close()
	if err := postgres.MigrateCatalog(startupContext, db); err != nil {
		cancel()
		log.Fatal(err)
	}
	cancel()

	brokers := strings.Split(envOrDefault("KAFKA_BROKERS", "localhost:9092"), ",")
	topic := envOrDefault("CATALOG_EVENTS_TOPIC", "catalog.product-events.v1")
	metrics := &eventflow.Metrics{}
	publisher, err := eventflow.NewPublisher(brokers, topic, postgres.NewOutboxStore(db), metrics)
	if err != nil {
		log.Fatal(err)
	}
	defer publisher.Close()

	service := catalogapp.NewService(postgres.NewCatalogStore(db))
	api := httpapi.NewCatalogHandler(service, func(ctx context.Context) error {
		if err := db.PingContext(ctx); err != nil {
			return err
		}
		pingContext, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		return publisher.Ping(pingContext)
	})
	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics)
	mux.Handle("/", api)
	log.Printf("catalog-service listening on %s", address)
	if err := httpserver.RunWithWorkers(address, mux, publisher.Run); err != nil {
		log.Fatal(err)
	}
}

func envOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
