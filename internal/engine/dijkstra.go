package engine

import (
	"errors"
	"time"

	"github.com/saviotito/currency-router/internal/models"
	"github.com/shopspring/decimal"
)

type Router struct {
	Graph *models.Graph
}

func NewRouter(g *models.Graph) *Router {
	return &Router{Graph: g}
}

func (r *Router) FindBestRoute(from, to string, amount decimal.Decimal) (models.CalculateResponse, error) {
	maxBalances := make(map[string]decimal.Decimal)
	parentEdge := make(map[string]models.Rate) // Tracks the edge we took to get to that max balance (for path reconstruction)

	for node := range r.Graph.Edges {
		maxBalances[node] = decimal.Zero
	} // Initialize all balances to 0

	for _, edges := range r.Graph.Edges {
		for _, edge := range edges {
			maxBalances[edge.To] = decimal.Zero
		}
	}

	// Starting point
	maxBalances[from] = amount

	// For simplicity in this step, we use a basic Dijkstra loop.
	// In a high-traffic app, you'd use a Priority Queue (Heap).
	visited := make(map[string]bool)

	for i := 0; i < len(maxBalances); i++ {
		// Find the unvisited currency with the HIGHEST current balance
		curr := ""
		maxVal := decimal.NewFromInt(-1)
		for c, bal := range maxBalances {
			if !visited[c] && bal.GreaterThan(maxVal) {
				maxVal = bal
				curr = c
			}
		}

		if curr == "" || maxVal.LessThanOrEqual(decimal.Zero) {
			break
		}

		visited[curr] = true

		// Explore neighbors
		for _, edge := range r.Graph.Edges[curr] {
			// THE SMART CALCULATION: Apply the fee and rate
			newBalance := edge.Apply(maxBalances[curr])

			// If this path gives us more money than previously found, update it
			if newBalance.GreaterThan(maxBalances[edge.To]) {
				maxBalances[edge.To] = newBalance
				parentEdge[edge.To] = edge
			}
		}
	}

	// Reconstruct the path from 'to' back to 'from'
	if maxBalances[to].IsZero() {
		return models.CalculateResponse{}, errors.New("no profitable path found")
	}

	return r.reconstruct(parentEdge, from, to, maxBalances[to]), nil
} // FindBestRoute calculates the path that results in the highest final amount.

func (r *Router) reconstruct(parentEdge map[string]models.Rate, from, to string, finalAmount decimal.Decimal) models.CalculateResponse {
	var path []models.Rate
	curr := to

	for curr != from {
		edge := parentEdge[curr]
		// Prepend to path
		path = append([]models.Rate{edge}, path...)
		curr = edge.From
	}

	return models.CalculateResponse{
		Path:        path,
		FinalAmount: finalAmount,
	}
}

//************ DIRECT *************

// GetDirectRoute picks the single-hop edge that delivers the most after
// fees, across all providers that quote this corridor. Returns (received,
// fee in source currency).
func (r *Router) GetDirectRoute(from, to string, amount decimal.Decimal) (decimal.Decimal, decimal.Decimal) {
	best := decimal.Zero
	bestFee := decimal.Zero
	for _, edge := range r.Graph.Edges[from] {
		if edge.To != to {
			continue
		}
		out := edge.Apply(amount)
		if out.GreaterThan(best) {
			best = out
			bestFee = edge.FeeFlat.Add(amount.Mul(edge.FeePercentage))
		}
	}
	return best, bestFee
}

// GetDirectMidMarket returns amount * the best direct-edge rate (no fee
// deduction) across providers quoting the corridor — the figure a Wise-
// style "mid-market" widget would show. Zero if no direct edge exists.
func (r *Router) GetDirectMidMarket(from, to string, amount decimal.Decimal) decimal.Decimal {
	best := decimal.Zero
	for _, edge := range r.Graph.Edges[from] {
		if edge.To != to {
			continue
		}
		mid := amount.Mul(edge.Value)
		if mid.GreaterThan(best) {
			best = mid
		}
	}
	return best
}

func CalculateConfidence(path []models.Rate) int {
	if len(path) == 0 {
		return 0
	}

	score := 100
	now := time.Now()

	for _, edge := range path {
		diff := now.Sub(edge.LastUpdate).Minutes()
		if diff > 2 {
			score -= int(diff * 5) // Drop 5 points per minute over 2 mins
		}
	}

	if score < 0 {
		return 0
	}
	return score
}
