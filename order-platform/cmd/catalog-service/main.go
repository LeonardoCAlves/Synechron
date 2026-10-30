package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/LeonardoCAlves/synechron/order-platform/internal/catalogapp"
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

	service := catalogapp.NewService(postgres.NewCatalogStore(db))
	handler := httpapi.NewCatalogHandler(service)
	log.Printf("catalog-service listening on %s", address)
	if err := httpserver.Run(address, handler); err != nil {
		log.Fatal(err)
	}
}
