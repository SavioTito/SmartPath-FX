package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Rate struct {
	From          string          `json:"from"`
	To            string          `json:"to"`
	Value         decimal.Decimal `json:"value"`
	FeeFlat       decimal.Decimal `json:"fee_flat"`
	FeePercentage decimal.Decimal `json:"fee_percentage"`
	FeeCurrency   string          `json:"fee_currency"`
	Provider      string          `json:"provider"`
	LastUpdate    time.Time       `json:"last_update"`
}

// Apply returns the net amount in the destination currency after the
// provider deducts its fee. Fee = flat + percentage * amount, where flat is
// assumed denominated in the source currency (FeeCurrency). The handler is
// responsible for normalizing fees on hops where FeeCurrency != From.
func (r Rate) Apply(amount decimal.Decimal) decimal.Decimal {
	fee := r.FeeFlat.Add(amount.Mul(r.FeePercentage))
	net := amount.Sub(fee)
	if net.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero
	}
	return net.Mul(r.Value)
}

type Graph struct {
	Edges map[string][]Rate
}

func NewRate(from, to string, value, feeFlat, feePercentage decimal.Decimal, feeCurrency, provider string) Rate {
	return Rate{
		From:          from,
		To:            to,
		Value:         value,
		FeeFlat:       feeFlat,
		FeePercentage: feePercentage,
		FeeCurrency:   feeCurrency,
		Provider:      provider,
		LastUpdate:    time.Now(),
	}
}

func NewGraph() *Graph {
	return &Graph{
		Edges: make(map[string][]Rate),
	}
}

func (g *Graph) AddRate(r Rate) {
	g.Edges[r.From] = append(g.Edges[r.From], r)
}
