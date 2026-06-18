package engine

import (
	"testing"

	"github.com/saviotito/currency-router/internal/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func makeRate(from, to string, value, feePct float64) models.Rate {
	return models.NewRate(
		from, to,
		decimal.NewFromFloat(value),
		decimal.Zero,
		decimal.NewFromFloat(feePct),
		from, "test",
	)
}

func buildGraph(rates ...models.Rate) *models.Graph {
	g := models.NewGraph()
	for _, r := range rates {
		g.AddRate(r)
	}
	return g
}

func containsCurrency(path []models.Rate, code string) bool {
	for _, e := range path {
		if e.From == code || e.To == code {
			return true
		}
	}
	return false
}

func TestNoArbitrage_LinearGraph(t *testing.T) {
	g := buildGraph(
		makeRate("USD", "EUR", 0.9, 0),
		makeRate("EUR", "GBP", 0.8, 0),
	)
	cycles, err := NewArbitrageDetector(g).DetectCycles()
	assert.NoError(t, err)
	assert.Empty(t, cycles)
}

func TestNoArbitrage_BalancedCycle(t *testing.T) {
	// Effective product = 1.0 exactly; below threshold.
	g := buildGraph(
		makeRate("USD", "EUR", 0.9, 0),
		makeRate("EUR", "GBP", 0.8, 0),
		makeRate("GBP", "USD", 1.0/(0.9*0.8), 0),
	)
	cycles, err := NewArbitrageDetector(g).DetectCycles()
	assert.NoError(t, err)
	assert.Empty(t, cycles)
}

func TestArbitrage_SimpleCycle(t *testing.T) {
	// Product = 0.9 * 0.8 * 1.458333... = 1.05
	g := buildGraph(
		makeRate("USD", "EUR", 0.9, 0),
		makeRate("EUR", "GBP", 0.8, 0),
		makeRate("GBP", "USD", 1.05/(0.9*0.8), 0),
	)
	cycles, err := NewArbitrageDetector(g).DetectCycles()
	assert.NoError(t, err)
	assert.Len(t, cycles, 1)

	expected := decimal.NewFromFloat(1.05)
	diff := cycles[0].ProfitFactor.Sub(expected).Abs()
	assert.True(t, diff.LessThan(decimal.NewFromFloat(0.001)),
		"profit factor %s not within 0.001 of 1.05", cycles[0].ProfitFactor)
}

func TestArbitrage_MultipleCycles(t *testing.T) {
	g := buildGraph(
		// Cycle 1: USD → EUR → GBP → USD, product ~1.03
		makeRate("USD", "EUR", 0.9, 0),
		makeRate("EUR", "GBP", 0.8, 0),
		makeRate("GBP", "USD", 1.03/(0.9*0.8), 0),

		// Cycle 2: JPY → CAD → AUD → JPY, product ~1.03
		makeRate("JPY", "CAD", 0.01, 0),
		makeRate("CAD", "AUD", 1.2, 0),
		makeRate("AUD", "JPY", 1.03/(0.01*1.2), 0),
	)
	cycles, err := NewArbitrageDetector(g).DetectCycles()
	assert.NoError(t, err)
	assert.Len(t, cycles, 2)
}

func TestArbitrage_FeesEliminateProfit(t *testing.T) {
	// Raw rates multiply to 1.03 but per-edge FeePercentage 0.015
	// reduces effective product below the 1.0001 threshold.
	// effective product = 1.03 * (1 - 0.015)^3 ≈ 0.9844 → empty
	g := buildGraph(
		makeRate("USD", "EUR", 0.9, 0.015),
		makeRate("EUR", "GBP", 0.8, 0.015),
		makeRate("GBP", "USD", 1.03/(0.9*0.8), 0.015),
	)
	cycles, err := NewArbitrageDetector(g).DetectCycles()
	assert.NoError(t, err)
	assert.Empty(t, cycles)
}

func TestDetectFromCurrency_Scoped(t *testing.T) {
	g := buildGraph(
		// Cycle touching USD
		makeRate("USD", "EUR", 0.9, 0),
		makeRate("EUR", "GBP", 0.8, 0),
		makeRate("GBP", "USD", 1.03/(0.9*0.8), 0),

		// Cycle NOT touching USD
		makeRate("JPY", "CAD", 0.01, 0),
		makeRate("CAD", "AUD", 1.2, 0),
		makeRate("AUD", "JPY", 1.03/(0.01*1.2), 0),
	)
	d := NewArbitrageDetector(g)
	all, err := d.DetectCycles()
	assert.NoError(t, err)
	assert.Len(t, all, 2)

	usdOnly, err := d.DetectFromCurrency("USD")
	assert.NoError(t, err)
	assert.Len(t, usdOnly, 1)
	assert.True(t, containsCurrency(usdOnly[0].Path, "USD"))
}

func TestDedupe_RotatedCycles(t *testing.T) {
	// Triangular cycle with a fourth node feeding into multiple corners,
	// so Bellman-Ford's relaxation pass can taint more than one node in
	// the cycle and reconstruct rotated views of the same loop.
	g := buildGraph(
		makeRate("USD", "EUR", 0.9, 0),
		makeRate("EUR", "GBP", 0.8, 0),
		makeRate("GBP", "USD", 1.05/(0.9*0.8), 0),

		// Feeders ensure each cycle node is reachable from elsewhere,
		// triggering multiple relaxations on the V-th pass.
		makeRate("CHF", "USD", 1.1, 0),
		makeRate("CHF", "EUR", 0.95, 0),
		makeRate("CHF", "GBP", 0.85, 0),
	)
	cycles, err := NewArbitrageDetector(g).DetectCycles()
	assert.NoError(t, err)
	assert.Len(t, cycles, 1, "rotated reconstructions of the same cycle must dedupe to one entry")
}
