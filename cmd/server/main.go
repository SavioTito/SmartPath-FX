package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/saviotito/currency-router/internal/engine"
	"github.com/saviotito/currency-router/internal/models"
	"github.com/saviotito/currency-router/internal/providers"
	"github.com/saviotito/currency-router/internal/version"
	"github.com/shopspring/decimal"
)

func main() {
	decimal.MarshalJSONWithoutQuotes = true

	wise := providers.NewWiseProvider()

	providerList := []models.ExchangeProvider{wise}

	aggregator := engine.NewAggregator(providerList)
	cache := engine.NewMemoryCache()

	handler := &engine.RouterHandler{
		Aggregator: aggregator,
		Cache:      cache,
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	mux := http.NewServeMux()
	mux.Handle("/calculate", handler)
	mux.Handle("POST /arbitrage/scan", &engine.ArbitrageScanHandler{
		Fetcher: aggregator,
		Cache:   cache,
	})
	mux.Handle("GET /arbitrage/from/{currency}", &engine.ArbitrageFromHandler{
		Fetcher: aggregator,
		Cache:   cache,
	})
	mux.Handle("GET /healthz", &engine.HealthHandler{
		StartTime: time.Now(),
		Providers: providerList,
		Cache:     cache,
	})

	log.Printf("SmartPath-FX %s listening on :%s (graph TTL 5m)", version.Version, port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
