package engine

import (
	"fmt"

	"github.com/saviotito/currency-router/internal/models"
)

type Aggregator struct {
	providers []models.ExchangeProvider
} // Aggregator coordinates multiple providers to build the graph.

func NewAggregator(providers []models.ExchangeProvider) *Aggregator {
	return &Aggregator{
		providers: providers,
	}
}

func (a *Aggregator) FetchAll(base string) *models.Graph {
	graph := models.NewGraph()
	results := make(chan []models.Rate, len(a.providers))

	for _, p := range a.providers {
		go func(prov models.ExchangeProvider) {
			fmt.Printf("DEBUG: Aggregator calling provider: %s\n", prov.Name())

			rates, err := prov.FetchRates(base)
			if err != nil {
				fmt.Printf("ERROR: Provider %s failed: %v\n", prov.Name(), err)
				results <- nil
				return
			}
			results <- rates
		}(p) // Launch a goroutine for each provider
	}

	for i := 0; i < len(a.providers); i++ {
		rates := <-results
		fmt.Printf("DEBUG: Received %d rates from a provider\n", len(rates))
		for _, r := range rates {
			graph.AddRate(r)
		}
	} // Collect the results

	return graph

}
