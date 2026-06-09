package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/saviotito/currency-router/internal/engine"
	"github.com/saviotito/currency-router/internal/models"
	"github.com/saviotito/currency-router/internal/providers"
	"github.com/shopspring/decimal"
)

func main() {
	decimal.MarshalJSONWithoutQuotes = true

	fmt.Println("--- Smart Currency Router Engine Starting ---")

	wise := providers.NewWiseProvider()

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
