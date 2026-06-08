package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Rate struct {
	From        string          `json:"from"`
	To          string          `json:"to"`
	Value       decimal.Decimal `json:"value"`
	FixedFee    decimal.Decimal `json:"fixed_fee"`
	FeeCurrency string          `json:"fee_currency"`
	Provider    string          `json:"provider"`
	LastUpdate  time.Time       `json:"last_update"`
}

func (r Rate) Apply(amount decimal.Decimal) decimal.Decimal {
	if amount.LessThanOrEqual(r.FixedFee) {
		return decimal.Zero
	}
	return amount.Sub(r.FixedFee).Mul(r.Value)
} // Calculates how much money is left after crossing this edge.

type Graph struct {
	Edges map[string][]Rate
}

func NewRate(from, to string, value, fixedFee decimal.Decimal, provider string) Rate {
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
