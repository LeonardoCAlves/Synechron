package main

import (
	"context"
	"log"
	"os"
	"time"

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

	store := postgres.NewOrderStore(db)
	service := orderapp.NewService(store, store)
	handler := httpapi.NewOrderHandler(service)
	log.Printf("order-service listening on %s", address)
	if err := httpserver.Run(address, handler); err != nil {
		log.Fatal(err)
	}
}
