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

	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, fetching from system environment")
	}

	fmt.Println("--- Currency Router Engine Starting ---")

	apiKey := os.Getenv("WISE_API_KEY")
	apiURL := os.Getenv("WISE_API_URL")
	if apiKey == "" || apiURL == "" {
		log.Fatal("Missing environment variables: WISE_API_KEY or WISE_API_URL")
	}

	wise := providers.NewWiseProvider(apiKey, apiURL)
	agg := engine.NewAggregator([]models.ExchangeProvider{wise})
	cache := engine.NewMemoryCache()

	h := &engine.RouterHandler{
		Aggregator: agg,
		Cache:      cache,
	}
	mux := http.NewServeMux()

	mux.Handle("/v1/calculate", h) // Register the endpoint

	fmt.Println("Server listening on:8080...")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
