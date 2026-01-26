package main

import (
	"fmt"
	"log"

	"github.com/saviotito/currency-router/internal/engine"
	"github.com/saviotito/currency-router/internal/models"
	"github.com/saviotito/currency-router/internal/providers"
)

func main() {
	fmt.Println("--- Currency Router Engine Starting ---")

	wise := providers.NewWiseProvider("YOUR_FAKE_API_KEY")

	providersList := []models.ExchangeProvider{wise}
	agg := engine.NewAggregator(providersList)

	fmt.Println("Connecting to providers...")
	graph := agg.FetchAll("USD") //Fetch all rates starting from USD

	path, err := engine.FindBestRoute(graph, "USD", "EUR")
	if err != nil {
		log.Printf("Route Calculation Failed: %v", err)
		fmt.Println("Tip: This is expected if the API returned 401 Unauthorized.")
		return
	}

	fmt.Printf("Optimal Path Found: %v", path)

}
