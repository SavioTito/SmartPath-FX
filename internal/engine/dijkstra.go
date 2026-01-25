package engine

import (
	"errors"
	"math"

	"github.com/saviotito/currency-router/internal/models"
)

func FindBestRoute(graph *models.Graph, start, end string) ([]models.Rate, error) {
	distances := make(map[string]float64)
	previous := make(map[string]*models.Rate)
	visited := make(map[string]bool)

	for node := range graph.Edges {
		distances[node] = math.Inf(1)
	} // Initialize distances to Infinity

	distances[end] = math.Inf(1) // The target node might not be a source node in the map, so add it manually
	distances[start] = 0

	for i := 0; i < len(distances); i++ {
		u := ""
		minDist := math.Inf(1)

		for node, dist := range distances {
			if !visited[node] && dist < minDist {
				minDist = dist
				u = node
			}
		} // Find the unvisited node with the smallest distance

		if u == "" || u == end {
			break
		}
		visited[u] = true

		for _, edge := range graph.Edges[u] {
			newDist := distances[u] + edge.Weight
			if newDist < distances[edge.To] {
				distances[edge.To] = newDist // We store the edge itself so we know which provider gave us this rate
				edgeCopy := edge
				previous[edge.To] = &edgeCopy
			}
		}
	}

	var path []models.Rate
	curr := end
	for previous[curr] != nil {
		edge := previous[curr]
		path = append([]models.Rate{*edge}, path...)
		curr = edge.From
	}

	if len(path) == 0 && start != end {
		return nil, errors.New("No route found")
	}

	return path, nil
} //Calculates the optimal path between start and end currencies.
