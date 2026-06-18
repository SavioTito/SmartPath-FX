package engine

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/saviotito/currency-router/internal/models"
)

// graphFetcher is satisfied by *Aggregator and test doubles.
type graphFetcher interface {
	FetchFullGraph(bases []string) *models.Graph
}

var defaultBases = []string{"USD", "EUR", "GBP", "BTC"}

type arbitrageScanResponse struct {
	Cycles      []ArbitrageCycle `json:"cycles"`
	ScannedAt   time.Time        `json:"scanned_at"`
	GraphSource string           `json:"graph_source"`
}

type arbitrageFromResponse struct {
	StartCurrency string           `json:"start_currency"`
	Cycles        []ArbitrageCycle `json:"cycles"`
	ScannedAt     time.Time        `json:"scanned_at"`
	GraphSource   string           `json:"graph_source"`
}

// ArbitrageScanHandler handles POST /arbitrage/scan.
type ArbitrageScanHandler struct {
	Fetcher graphFetcher
	Cache   *MemoryChace
}

func (h *ArbitrageScanHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var body struct {
		BaseCurrencies []string `json:"base_currencies"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err.Error() != "EOF" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	bases := body.BaseCurrencies
	if len(bases) == 0 {
		bases = defaultBases
	}

	sorted := make([]string, len(bases))
	copy(sorted, bases)
	sort.Strings(sorted)
	cacheKey := "arbitrage:full:" + strings.Join(sorted, "-")

	graph, found := h.Cache.Get(cacheKey)
	graphSource := "live"
	if found {
		graphSource = "cached"
	} else {
		fmt.Printf("Cache Missing: Building full arbitrage graph for %v...\n", sorted)
		graph = h.Fetcher.FetchFullGraph(bases)
		h.Cache.Set(cacheKey, graph, 5*time.Minute)
	}

	cycles, err := NewArbitrageDetector(graph).DetectCycles()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	if cycles == nil {
		cycles = []ArbitrageCycle{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(arbitrageScanResponse{
		Cycles:      cycles,
		ScannedAt:   time.Now(),
		GraphSource: graphSource,
	})
}

// ArbitrageFromHandler handles GET /arbitrage/from/{currency}.
type ArbitrageFromHandler struct {
	Fetcher graphFetcher
	Cache   *MemoryChace
}

func (h *ArbitrageFromHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	currency := strings.ToUpper(strings.TrimSpace(r.PathValue("currency")))
	if currency == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "currency path parameter required"})
		return
	}

	sorted := make([]string, len(defaultBases))
	copy(sorted, defaultBases)
	sort.Strings(sorted)
	cacheKey := "arbitrage:full:" + strings.Join(sorted, "-")

	graph, found := h.Cache.Get(cacheKey)
	graphSource := "live"
	if found {
		graphSource = "cached"
	} else {
		fmt.Printf("Cache Missing: Building full arbitrage graph for %v...\n", sorted)
		graph = h.Fetcher.FetchFullGraph(defaultBases)
		h.Cache.Set(cacheKey, graph, 5*time.Minute)
	}

	cycles, err := NewArbitrageDetector(graph).DetectFromCurrency(currency)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	if cycles == nil {
		cycles = []ArbitrageCycle{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(arbitrageFromResponse{
		StartCurrency: currency,
		Cycles:        cycles,
		ScannedAt:     time.Now(),
		GraphSource:   graphSource,
	})
}
