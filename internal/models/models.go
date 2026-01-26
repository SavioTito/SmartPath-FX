package models

import (
	"time"
)

type Rate struct {
	From       string    `json:"from"`
	To         string    `json:"to"`
	Value      float64   `json:"value"`
	FixedFee   float64   `json:"fixed_fee"`
	Provider   string    `json:"provider"`
	LastUpdate time.Time `json:"last_update"`
}

func (r Rate) Apply(amount float64) float64 {
	if amount <= r.FixedFee {
		return 0
	}
	return (amount - r.FixedFee) * r.Value
} // Calculates how much money is left after crossing this edge.

type Graph struct {
	Edges map[string][]Rate
}

func NewRate(from, to string, value, fixedFee float64, provider string) Rate {
	return Rate{
		From:       from,
		To:         to,
		Value:      value,
		FixedFee:   fixedFee,
		Provider:   provider,
		LastUpdate: time.Now(),
	}
}

func NewGraph() *Graph {
	return &Graph{
		Edges: make(map[string][]Rate),
	}
} // NewGraph initializes the map to avoid pointer errors.

func (g *Graph) AddRate(r Rate) {
	g.Edges[r.From] = append(g.Edges[r.From], r)
}
