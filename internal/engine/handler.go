package engine

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/saviotito/currency-router/internal/models"
)

type RouterHandler struct {
	Aggregator *Aggregator
	Cache      *MemoryChace
}

func (h *RouterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.CalculateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Amount <= 0 {
		http.Error(w, "Amount must be greater than 0", http.StatusBadRequest)
		return
	}

	// Smart Cache Key:
	// Since the graph now includes bridges, the cache key should probably
	// include both Source and Target, as FetchSmartGraph is optimized for the pair.
	cacheKey := fmt.Sprintf("%s-%s", req.From, req.To)
	graph, found := h.Cache.Get(cacheKey)

	if !found {
		fmt.Printf("Cache Miss: Building Smart Graph for %s -> %s...\n", req.From, req.To)
		graph = h.Aggregator.FetchSmartGraph(req.From, req.To)
		h.Cache.Set(cacheKey, graph, 5*time.Minute)
	}

	// Use the New Router
	router := NewRouter(graph)
	response, err := router.FindBestRoute(req.From, req.To, req.Amount)

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	// Send the response
	// (The Router already calculated the final amount and path)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
