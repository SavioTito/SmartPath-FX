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
	var graphMu sync.Mutex

	bridges := []string{"USD", "EUR", "GBP", "BTC"}
	froms := dedup(append([]string{source}, bridges...))
	tos := dedup(append([]string{target}, bridges...))

	var wg sync.WaitGroup
	for _, p := range a.providers {
		for _, from := range froms {
			for _, to := range tos {
				if from == to {
					continue
				}
				wg.Add(1)
				go func(prov models.ExchangeProvider, from, to string) {
					defer wg.Done()
					quotes, err := prov.QuoteAllProviders(from, to)
					if err != nil {
						fmt.Printf("WARN: %s %s->%s quote failed, dropping edge group: %v\n", prov.Name(), from, to, err)
						return
					}
					edges := make([]models.Rate, 0, len(quotes))
					for _, q := range quotes {
						edges = append(edges, models.Rate{
							From:          from,
							To:            to,
							Value:         q.Rate,
							FeeFlat:       q.Flat,
							FeePercentage: q.Percentage,
							FeeCurrency:   q.Currency,
							Provider:      q.Provider,
							LastUpdate:    q.FetchedAt,
						})
					}
					graphMu.Lock()
					for _, e := range edges {
						graph.AddRate(e)
					}
					graphMu.Unlock()
				}(p, from, to)
			}
		}
	}

	wg.Wait()
	return graph
}

func dedup(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
