package engine

import (
	"fmt"
	"testing"

	"github.com/saviotito/currency-router/internal/models"
	"github.com/shopspring/decimal"
)

// buildNxMGraph builds an n-currency graph with edgesPerNode outgoing edges per node.
// Ring + skip edges guarantee connectivity from C000 to C(n-1).
func buildNxMGraph(n, edgesPerNode int) *models.Graph {
	codes := make([]string, n)
	for i := range codes {
		codes[i] = fmt.Sprintf("C%03d", i)
	}
	var rates []models.Rate
	for i := range n {
		for j := 1; j <= edgesPerNode && i+j < n; j++ {
			rate := 1.0 + float64((i*edgesPerNode+j)%100)*0.001
			rates = append(rates, makeRate(codes[i], codes[i+j], rate, 0.001))
		}
	}
	return buildGraph(rates...)
}

// buildCycleGraph builds an n-currency ring graph with one profitable triangle injected
// on C000→C001→C002→C000 (rate 1.1 per leg, zero fee → product 1.331 > threshold).
func buildCycleGraph(n int) *models.Graph {
	codes := make([]string, n)
	for i := range codes {
		codes[i] = fmt.Sprintf("C%03d", i)
	}
	var rates []models.Rate
	for i := range n {
		next := (i + 1) % n
		rates = append(rates, makeRate(codes[i], codes[next], 1.0+float64(i)*0.0001, 0.001))
	}
	for i := 0; i < n; i += 2 {
		j := (i + n/2) % n
		if j != i {
			rates = append(rates, makeRate(codes[i], codes[j], 1.0005, 0.001))
		}
	}
	// profitable triangle: each leg 1.1, zero fee → product 1.331; bestEdge picks these
	// over ring's ~0.999 effective rates on the same corridors
	rates = append(rates,
		makeRate("C000", "C001", 1.1, 0),
		makeRate("C001", "C002", 1.1, 0),
		makeRate("C002", "C000", 1.1, 0),
	)
	return buildGraph(rates...)
}

// buildFakeQuotes returns count Rate objects spread across 5 providers × 10 corridors.
func buildFakeQuotes(count int) []models.Rate {
	providers := []string{"wise", "xe", "oanda", "kraken", "coinbase"}
	corridors := [][2]string{
		{"USD", "EUR"}, {"USD", "GBP"}, {"USD", "JPY"}, {"USD", "CAD"}, {"USD", "AUD"},
		{"EUR", "GBP"}, {"EUR", "JPY"}, {"EUR", "CHF"}, {"GBP", "JPY"}, {"GBP", "CAD"},
	}
	out := make([]models.Rate, count)
	for i := range out {
		c := corridors[i%len(corridors)]
		out[i] = models.NewRate(
			c[0], c[1],
			decimal.NewFromFloat(1.0+float64(i)*0.001),
			decimal.Zero,
			decimal.NewFromFloat(0.001),
			c[0],
			providers[i%len(providers)],
		)
	}
	return out
}

func BenchmarkFindBestRoute(b *testing.B) {
	for _, n := range []int{10, 50, 100} {
		b.Run(fmt.Sprintf("%d-currencies", n), func(b *testing.B) {
			g := buildNxMGraph(n, 5)
			r := NewRouter(g)
			src := "C000"
			dst := fmt.Sprintf("C%03d", n-1)
			amount := decimal.NewFromInt(1000)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				_, _ = r.FindBestRoute(src, dst, amount)
			}
		})
	}
}

func BenchmarkDetectCycles(b *testing.B) {
	for _, n := range []int{10, 30, 100} {
		b.Run(fmt.Sprintf("%d-currencies", n), func(b *testing.B) {
			g := buildCycleGraph(n)
			d := NewArbitrageDetector(g)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				_, _ = d.DetectCycles()
			}
		})
	}
}

func BenchmarkAggregatorMerge(b *testing.B) {
	quotes := buildFakeQuotes(50)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		g := models.NewGraph()
		for _, q := range quotes {
			g.AddRate(q)
		}
	}
}
