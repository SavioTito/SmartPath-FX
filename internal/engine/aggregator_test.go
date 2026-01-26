package engine

import (
	"testing"

	"github.com/saviotito/currency-router/internal/models"
)

type MockProvider struct {
	NameStr string
	Rates   []models.Rate
} //Fake API for testing

func (m MockProvider) Name() string {
	return m.NameStr
}

func (m MockProvider) FetchRates(base string) ([]models.Rate, error) {
	return m.Rates, nil
}

func TestAggregator_FetchAll(t *testing.T) {
	p1 := MockProvider{
		NameStr: "Provider1",
		Rates: []models.Rate{
			models.NewRate("USD", "EUR", 0.92, "Provider1"),
		},
	}

	p2 := MockProvider{
		NameStr: "Provider2",
		Rates: []models.Rate{
			models.NewRate("EUR", "AOA", 1000.0, "Provider2"),
		},
	}

	agg := NewAggregator([]models.ExchangeProvider{p1, p2})

	graph := agg.FetchAll("USD")

	if len(graph.Edges["USD"]) == 0 {
		t.Error("Expected EUR rates from Provider2, but got none")
	}

	path, err := FindBestRoute(graph, "USD", "AOA")
	if err != nil {
		t.Fatalf("Aggregator failed to link providers: %v", err)
	} // Final Boss Check: Can Dijkstra find a path across two different providers?

	if len(path) != 2 {
		t.Errorf("Expected path length 2, got %d", len(path))
	}
}
