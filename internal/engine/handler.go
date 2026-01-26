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
	} // Only allow POST requests

	var req models.CalculateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	} //Decode the request

	graph, found := h.Cache.Get(req.From)
	if !found {
		fmt.Println("Cache Miss: Fetching from API...")
		graph = h.Aggregator.FetchAll(req.From)     //Fetch fresh data
		h.Cache.Set(req.From, graph, 1*time.Minute) //Cache the new data
	}

	path, err := FindBestRoute(graph, req.From, req.To)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "No viable currency route found",
		})
		return
	} //Calculate best route

	total := 1.0
	for _, step := range path {
		total *= step.Value
	} //Calculate total combined rate for the user

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.CalculateResponse{
		Path:  path,
		Total: total,
	}) //Send response

}
