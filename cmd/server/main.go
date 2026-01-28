package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/saviotito/currency-router/internal/engine"
	"github.com/saviotito/currency-router/internal/models"
	"github.com/saviotito/currency-router/internal/providers"
)

func main() {
	fmt.Println("--- Smart Currency Router Engine Starting ---")

	// Load Environment Variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system env")
	}

	apiKey := os.Getenv("WISE_API_KEY")
	apiURL := os.Getenv("WISE_API_URL")
	if apiKey == "" || apiURL == "" {
		log.Fatal("WISE_API_KEY or WISE_API_URL not set in environment")
	}
	wise := providers.NewWiseProvider(apiKey, apiURL)

	providerList := []models.ExchangeProvider{wise}

	aggregator := engine.NewAggregator(providerList)
	cache := engine.NewMemoryCache()

	handler := &engine.RouterHandler{
		Aggregator: aggregator,
		Cache:      cache,
	}

	port := ":8080"
	http.Handle("/calculate", handler)

	fmt.Printf("Server listening on %s...\n", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
