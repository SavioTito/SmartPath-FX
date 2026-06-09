package engine

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/saviotito/currency-router/internal/models"
	"github.com/shopspring/decimal"
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

	req.From = strings.ToUpper(strings.TrimSpace(req.From))
	req.To = strings.ToUpper(strings.TrimSpace(req.To))
	if req.From == "" || req.To == "" {
		http.Error(w, "from and to must be set", http.StatusBadRequest)
		return
	}

	if req.Amount.LessThanOrEqual(decimal.Zero) {
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

	// --- PRECISION AUDIT: All intermediate math runs in decimal.Decimal ---
	smartAmountRaw := smartRoute.FinalAmount
	directAmountRaw, _ := router.GetDirectRoute(req.From, req.To, req.Amount)
	directMidMarketRaw := router.GetDirectMidMarket(req.From, req.To, req.Amount)

	// Build currency -> cumulative-rate-from-source map by walking the path.
	// nodeRate[c] = how many units of c equal 1 unit of request.From, derived
	// from the chosen path's edge values. Lets us normalize a fee in ANY
	// path-currency back to the request source, instead of assuming
	// FeeCurrency == edge.From.
	nodeRate := map[string]decimal.Decimal{
		smartRoute.Path[0].From: decimal.NewFromInt(1),
	}
	cum := decimal.NewFromInt(1)
	for _, edge := range smartRoute.Path {
		cum = cum.Mul(edge.Value)
		nodeRate[edge.To] = cum
	}

	// Replay the path forward to learn the actual amount held at each hop, then
	// compute that hop's real fee (flat + pct * hopAmount). Fees are then
	// normalized back to the source currency via nodeRate.
	totalFeesSource := decimal.Zero
	hopAmount := req.Amount
	for _, edge := range smartRoute.Path {
		feeCur := edge.FeeCurrency
		if feeCur == "" {
			feeCur = edge.From
		}
		rate, ok := nodeRate[feeCur]
		if !ok {
			fmt.Printf("WARN: fee currency %s for %s->%s not on path, skipping fee normalization\n",
				feeCur, edge.From, edge.To)
			hopAmount = edge.Apply(hopAmount)
			continue
		}
		hopFee := edge.FeeFlat.Add(hopAmount.Mul(edge.FeePercentage))
		totalFeesSource = totalFeesSource.Add(hopFee.Div(rate))
		hopAmount = edge.Apply(hopAmount)
	}

	savingsRaw := smartAmountRaw.Sub(directAmountRaw)

	efficiencyTag := "Standard"
	savingsPct := decimal.Zero
	if directAmountRaw.GreaterThan(decimal.Zero) {
		savingsPct = savingsRaw.Div(directAmountRaw).Mul(decimal.NewFromInt(100))

		// If the difference is effectively zero (less than 0.001%), it's High Efficiency
		if savingsPct.Abs().LessThan(decimal.NewFromFloat(0.001)) {
			efficiencyTag = "High Efficiency"
		}

		savingsPct = savingsPct.Round(4)
	}

	smartMidMarketRaw := req.Amount.Mul(nodeRate[req.To])

	finalResponse := models.ProductionResponse{
		Request: req,
		Summary: models.CalculateSummary{
			SmartFinalAmount:      models.RoundToTwo(smartAmountRaw),
			SmartMidMarketAmount:  models.RoundToTwo(smartMidMarketRaw),
			DirectFinalAmount:     models.RoundToTwo(directAmountRaw),
			DirectMidMarketAmount: models.RoundToTwo(directMidMarketRaw),
			TotalSavings:          models.RoundToTwo(savingsRaw),
			SavingsPercentage:     savingsPct,
			TotalFixedFeesSource:  models.RoundToTwo(totalFeesSource),
		},
		SmartPath: smartRoute.Path,
		Meta: models.Metadata{
			ConfidenceScore: calculateConfidence(smartRoute.Path),
			Timestamp:       time.Now(),
			Efficiency:      efficiencyTag,
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
