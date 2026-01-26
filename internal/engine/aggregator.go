package engine

import (
	"net/http"
	"time"

	"github.com/saviotito/currency-router/internal/models"
)

type Aggregator struct {
	providers []models.ExchangeProvider
	client    *http.Client
} // Aggregator coordinates multiple providers to build the graph.

func NewAggregator(providers []models.ExchangeProvider) *Aggregator {
	return &Aggregator{
		providers: providers,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (a *Aggregator) FetchAll(base string) *models.Graph {
	graph := models.NewGraph()
	results := make(chan []models.Rate)

	for _, p := range a.providers {
		go func(prov models.ExchangeProvider) {
			rates, err := prov.FetchRates(base)
			if err != nil {
				results <- nil
				return
			}
			results <- rates
		}(p) // Launch a goroutine for each provider
	}

	for i := 0; i < len(a.providers); i++ {
		rates := <-results
		for _, r := range rates {
			graph.AddRate(r)
		}
	} // Collect the results

	return graph

}
