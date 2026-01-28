package models

import (
	"math"
	"time"
)

type CalculateRequest struct {
	From   string  `json:"from"`
	To     string  `json:"to"`
	Amount float64 `json:"amount"`
} // What we get from users

type CalculateResponse struct {
	Path        []Rate  `json:"path"`
	FinalAmount float64 `json:"final_amount"`
} //What we send back to them

type CalculateSummary struct {
	SmartFinalAmount     float64 `json:"smart_final_amount"`
	DirectFinalAmount    float64 `json:"direct_final_amount"`
	TotalSavings         float64 `json:"total_savings"`
	SavingsPercentage    float64 `json:"savings_percentage"`
	TotalFixedFeesSource float64 `json:"total_fixed_fees_source_currency"`
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

// RoundToTwo rounds floats to 2 decimal places for financial reporting
func RoundToTwo(val float64) float64 {
	return math.Round(val*100) / 100
}
