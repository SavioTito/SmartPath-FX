package engine

import (
	"errors"

	"github.com/saviotito/currency-router/internal/models"
)

type Router struct {
	Graph *models.Graph
}

func NewRouter(g *models.Graph) *Router {
	return &Router{Graph: g}
}

func (r *Router) FindBestRoute(from, to string, amount float64) (models.CalculateResponse, error) {
	maxBalances := make(map[string]float64)
	parentEdge := make(map[string]models.Rate) // Tracks the edge we took to get to that max balance (for path reconstruction)

	for node := range r.Graph.Edges {
		maxBalances[node] = 0
	} // Initialize all balances to 0

	for _, edges := range r.Graph.Edges {
		for _, edge := range edges {
			maxBalances[edge.To] = 0
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
		maxVal := -1.0
		for c, bal := range maxBalances {
			if !visited[c] && bal > maxVal {
				maxVal = bal
				curr = c
			}
		}

		if curr == "" || maxVal <= 0 {
			break
		}

		visited[curr] = true

		// Explore neighbors
		for _, edge := range r.Graph.Edges[curr] {
			// THE SMART CALCULATION: Apply the fee and rate
			newBalance := edge.Apply(maxBalances[curr])

			// If this path gives us more money than previously found, update it
			if newBalance > maxBalances[edge.To] {
				maxBalances[edge.To] = newBalance
				parentEdge[edge.To] = edge
			}
		}
	}

	// Reconstruct the path from 'to' back to 'from'
	if maxBalances[to] == 0 {
		return models.CalculateResponse{}, errors.New("no profitable path found")
	}

	return r.reconstruct(parentEdge, from, to, maxBalances[to]), nil
} // FindBestRoute calculates the path that results in the highest final amount.

func (r *Router) reconstruct(parentEdge map[string]models.Rate, from, to string, finalAmount float64) models.CalculateResponse {
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
