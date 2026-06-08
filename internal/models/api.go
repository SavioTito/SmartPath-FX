package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type CalculateRequest struct {
	From   string          `json:"from"`
	To     string          `json:"to"`
	Amount decimal.Decimal `json:"amount"`
} // What we get from users

type CalculateResponse struct {
	Path        []Rate          `json:"path"`
	FinalAmount decimal.Decimal `json:"final_amount"`
} //What we send back to them

type CalculateSummary struct {
	SmartFinalAmount     decimal.Decimal `json:"smart_final_amount"`
	DirectFinalAmount    decimal.Decimal `json:"direct_final_amount"`
	TotalSavings         decimal.Decimal `json:"total_savings"`
	SavingsPercentage    decimal.Decimal `json:"savings_percentage"`
	TotalFixedFeesSource decimal.Decimal `json:"total_fixed_fees_source_currency"`
}

type Metadata struct {
	ConfidenceScore int       `json:"confidence_score"`
	Timestamp       time.Time `json:"timestamp"`
	Efficiency      string    `json:"efficiency"`
}

type ProductionResponse struct {
	Request   CalculateRequest `json:"request"`
	Summary   CalculateSummary `json:"summary"`
	SmartPath []Rate           `json:"smart_path"`
	Meta      Metadata         `json:"meta"`
}

// RoundToTwo rounds decimals to 2 decimal places for financial reporting
func RoundToTwo(val decimal.Decimal) decimal.Decimal {
	return val.Round(2)
}
