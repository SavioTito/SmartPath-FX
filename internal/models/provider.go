package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type ExchangeProvider interface {
	QuoteFee(source, target string, amount decimal.Decimal) (FeeQuote, error)
	QuoteAllProviders(source, target string) (map[string]FeeQuote, error) // alias-keyed: "wise", "xe", ...
	Name() string
}

// FeeQuote carries rate + fee schedule for a corridor on a single provider.
// Provider is the human display name ("Wise", "Western Union"); the map key
// returned by QuoteAllProviders is the lowercase alias.
type FeeQuote struct {
	Provider   string
	Rate       decimal.Decimal // mid-market FX rate at probe time
	Flat       decimal.Decimal
	Percentage decimal.Decimal // 0.004 == 0.4%
	Currency   string          // currency the flat fee is denominated in
	Source     string          // "live" | "cached" | "fallback"
	FetchedAt  time.Time
}
