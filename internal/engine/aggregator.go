package engine

import (
	"fmt"
	"sync"

	"github.com/saviotito/currency-router/internal/models"
)

type Aggregator struct {
	providers []models.ExchangeProvider
}

func NewAggregator(providers []models.ExchangeProvider) *Aggregator {
	return &Aggregator{
		providers: providers,
	}
}

func (a *Aggregator) FetchSmartGraph(source, target string) *models.Graph {
	graph := models.NewGraph()

	bridges := []string{"USD", "EUR", "GBP", "BTC"}
	workList := append([]string{source}, bridges...)
	var wg sync.WaitGroup

	for _, p := range a.providers {
		for _, base := range workList {
			wg.Add(1)
			go func(prov models.ExchangeProvider, b string) {
				defer wg.Done()

				rates, err := prov.FetchRates(b)
				if err != nil {
					// Log and continue so one failing provider doesn't kill the whole search.
					fmt.Printf("WARN: Provider %s failed for base %s: %v\n", prov.Name(), b, err)
					return
				}

				for _, r := range rates {
					// Optimization: only store edges that lead to a bridge or the final target.
					if a.isRelevant(r.To, target, bridges) {
						graph.AddRate(r)
					}
				}
			}(p, base)
		}
	}

	wg.Wait()
	return graph
}

func (a *Aggregator) isRelevant(to, target string, bridges []string) bool {
	if to == target {
		return true
	}
	for _, b := range bridges {
		if to == b {
			return true
		}
	}
	return false
}
