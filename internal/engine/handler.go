package engine

import (
	"encoding/json"
	"fmt"
	"math"
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

	// Cache Management
	cacheKey := fmt.Sprintf("%s-%s", req.From, req.To)
	graph, found := h.Cache.Get(cacheKey)

	if !found {
		fmt.Printf("Cache Missing: Building Smart Graph for %s -> %s...\n", req.From, req.To)
		graph = h.Aggregator.FetchSmartGraph(req.From, req.To)
		h.Cache.Set(cacheKey, graph, 5*time.Minute)
	}

	// Routing Logic
	router := NewRouter(graph)
	smartRoute, err := router.FindBestRoute(req.From, req.To, req.Amount)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	// --- PRECISION AUDIT: Keep raw float64 for all intermediate logic ---
	smartAmountRaw := smartRoute.FinalAmount
	directAmountRaw, _ := router.GetDirectRoute(req.From, req.To, req.Amount)

	// Summarize Fees
	var totalFeesSource float64
	for i, edge := range smartRoute.Path {
		if i == 0 {
			totalFeesSource += edge.FixedFee
		} else {
			cumulativeRate := 1.0
			for j := 0; j < i; j++ {
				cumulativeRate *= smartRoute.Path[j].Value
			}
			totalFeesSource += (edge.FixedFee / cumulativeRate)
		}
	}

	// Savings Calculation
	savingsRaw := smartAmountRaw - directAmountRaw

	// Task 4: Efficiency Check (Buffer of 0.001%)
	efficiencyTag := "Standard"
	savingsPct := 0.0
	if directAmountRaw > 0 {
		savingsPct = (savingsRaw / directAmountRaw) * 100

		// If the difference is effectively zero (less than 0.001%), it's High Efficiency
		if math.Abs(savingsPct) < 0.001 {
			efficiencyTag = "High Efficiency"
		}

		// Precision: 4 decimal places for the percentage
		savingsPct = math.Round(savingsPct*10000) / 10000
	}

	// --- FINAL ROUNDING: Only happens here inside the JSON response builder ---
	finalResponse := models.ProductionResponse{
		Request: req,
		Summary: models.CalculateSummary{
			SmartFinalAmount:     models.RoundToTwo(smartAmountRaw),
			DirectFinalAmount:    models.RoundToTwo(directAmountRaw),
			TotalSavings:         models.RoundToTwo(savingsRaw),
			SavingsPercentage:    savingsPct,
			TotalFixedFeesSource: models.RoundToTwo(totalFeesSource),
		},
		SmartPath: smartRoute.Path,
		Meta: models.Metadata{
			ConfidenceScore: calculateConfidence(smartRoute.Path),
			Timestamp:       time.Now(),
			Efficiency:      efficiencyTag, // Ensure this exists in your models.Metadata struct!
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(finalResponse)
}

func calculateConfidence(path []models.Rate) int {
	if len(path) == 0 {
		return 0
	}
	score := 100
	now := time.Now()
	for _, edge := range path {
		minutesOld := now.Sub(edge.LastUpdate).Minutes()
		if minutesOld > 2 {
			score -= int(minutesOld-2) * 5
		}
	}
	if score < 0 {
		return 0
	}
	return score
}
