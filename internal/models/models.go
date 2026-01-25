package models

import (
	"math"
	"time"
)

type Rate struct {
	From       string    `json:"from"`
	To         string    `json:"to"`
	Value      float64   `json:"value"`
	Weight     float64   `json:"-"`
	Provider   string    `json:"provider"`
	LastUpdate time.Time `json:"last_update"`
} // Rate represents the connection between two currencies.

type Graph struct {
	Edges map[string][]Rate
}

func NewRate(from, to string, value float64, provider string) Rate {
	return Rate{
		From:       from,
		To:         to,
		Value:      value,
		Weight:     -math.Log(value),
		Provider:   provider,
		LastUpdate: time.Now(),
	}
} // NewRate is a constructor that handles the math.

func NewGraph() *Graph {
	return &Graph{
		Edges: make(map[string][]Rate),
	}
} // NewGraph initializes the map to avoid pointer errors.

func (g *Graph) AddRate(r Rate) {
	g.Edges[r.From] = append(g.Edges[r.From], r)
} // AddRate adds a rate to our graph.
