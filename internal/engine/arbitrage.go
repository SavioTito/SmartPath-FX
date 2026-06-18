// Package engine — arbitrage detection.
//
// # Why Bellman-Ford
//
// An arbitrage cycle in an FX graph is a sequence of trades whose product of
// effective rates exceeds 1: starting with one unit of currency C, you end
// up with more than one unit of C after traversing the cycle.
//
//	profit = r_1 * r_2 * ... * r_n > 1
//
// Taking the negative log of both sides turns the multiplicative test into
// an additive one:
//
//	-log(r_1) + -log(r_2) + ... + -log(r_n) < 0
//
// In other words, weighting each edge by -log(effRate) makes a profitable
// cycle equivalent to a negative-weight cycle in the resulting graph.
//
// Dijkstra's algorithm cannot find such cycles — it assumes non-negative
// edge weights and short-circuits as soon as it finds a "best" distance.
// Bellman-Ford does V-1 rounds of edge relaxation and uses a V-th round
// solely to detect cycles: any edge that still relaxes on that pass is part
// of (or reachable from) a negative cycle. This is the standard approach
// for FX arbitrage scanning.
//
// # Result reporting
//
// The algorithm runs in float64 (because math.Log requires it), but the
// reported ProfitFactor and ProfitPercent are recomputed in decimal.Decimal
// by multiplying the cycle edges' effective rates. Float is internal; the
// caller-facing answer is exact.
//
// # v1 simplification
//
// TODO(arbitrage-v2): Flat fees (Rate.FeeFlat) are ignored in v1. Cycle
// profitability is a property of rates, but flat fees are amount-dependent.
// Honouring them requires fixing a notional transfer amount and verifying
// the cycle stays profitable after each hop's flat-fee deduction — a
// different problem closer to min-cost flow than to negative-cycle
// detection. Add when an amount-parameterised API is needed.
package engine

import (
	"math"
	"sort"
	"strings"

	"github.com/saviotito/currency-router/internal/models"
	"github.com/shopspring/decimal"
)

// arbitrageThreshold is the minimum ProfitFactor a cycle must exceed to be
// reported. It exists to suppress floating-point noise from the math.Log
// round-trip — without it, a perfectly balanced cycle (product = 1.0) can
// occasionally surface as a phantom 1.0000000001 "opportunity".
const arbitrageThreshold = "1.0001"

// relaxEpsilon is the floor on a meaningful relaxation in float64 space.
// Without it, the V-th detection pass occasionally flags a "negative" cycle
// that is just ~1e-15 of fp drift from math.Log.
const relaxEpsilon = 1e-12

var thresholdDecimal = decimal.RequireFromString(arbitrageThreshold)

// ArbitrageDetector scans an FX Graph for profitable conversion cycles
// using Bellman-Ford negative-cycle detection on a log-transformed weight
// space.
//
// Example:
//
//	d := engine.NewArbitrageDetector(graph)
//	cycles, _ := d.DetectCycles()
type ArbitrageDetector struct {
	Graph *models.Graph
}

// ArbitrageCycle describes a single detected arbitrage opportunity.
type ArbitrageCycle struct {
	// Path is the cycle's edges in traversal order. Path[0].From ==
	// StartCurrency, Path[len-1].To == StartCurrency.
	Path []models.Rate `json:"path"`

	// StartCurrency is the cycle's canonical starting node (the
	// lexicographically smallest currency in the cycle, used for dedupe).
	StartCurrency string `json:"start_currency"`

	// ProfitFactor is the product of effective rates along the cycle —
	// always > arbitrageThreshold. A value of 1.05 means a 5 % round-trip
	// profit before flat fees.
	ProfitFactor decimal.Decimal `json:"profit_factor"`

	// ProfitPercent is (ProfitFactor - 1) * 100, pre-computed for callers.
	ProfitPercent decimal.Decimal `json:"profit_percent"`
}

// NewArbitrageDetector constructs a detector bound to the given Graph.
func NewArbitrageDetector(g *models.Graph) *ArbitrageDetector {
	return &ArbitrageDetector{Graph: g}
}

// DetectCycles returns every profitable arbitrage cycle in the graph,
// deduplicated so that rotations of the same cycle count once. Cycles are
// sorted by ProfitFactor in descending order.
//
// The returned error is reserved for forward compatibility; the v1
// implementation always returns nil. Empty graph, single node, disconnected
// graph, and "no arbitrage exists" all return an empty slice with nil error.
//
// Example:
//
//	cycles, _ := d.DetectCycles()
//	for _, c := range cycles { fmt.Println(c.StartCurrency, c.ProfitPercent) }
func (a *ArbitrageDetector) DetectCycles() ([]ArbitrageCycle, error) {
	if a == nil || a.Graph == nil || len(a.Graph.Edges) == 0 {
		return nil, nil
	}

	nodes, idx := buildNodeIndex(a.Graph)
	if len(nodes) < 2 {
		return nil, nil
	}

	edges := materializeEdges(a.Graph, idx)
	if len(edges) == 0 {
		return nil, nil
	}

	n := len(nodes)
	dist := make([]float64, n) // virtual super-source: all start at 0
	pred := make([]int, n)
	for i := range pred {
		pred[i] = -1
	}

	for i := 0; i < n-1; i++ {
		relaxed := false
		for _, e := range edges {
			if dist[e.u]+e.weight < dist[e.v]-relaxEpsilon {
				dist[e.v] = dist[e.u] + e.weight
				pred[e.v] = e.u
				relaxed = true
			}
		}
		if !relaxed {
			break
		}
	}

	tainted := make(map[int]struct{})
	for _, e := range edges {
		if dist[e.u]+e.weight < dist[e.v]-relaxEpsilon {
			tainted[e.v] = struct{}{}
		}
	}
	if len(tainted) == 0 {
		return nil, nil
	}

	seen := make(map[string]struct{})
	var out []ArbitrageCycle
	for v := range tainted {
		cycle := walkCycle(pred, v, n)
		if len(cycle) < 2 {
			continue
		}
		currencies := make([]string, len(cycle))
		for i, ni := range cycle {
			currencies[i] = nodes[ni]
		}
		path, ok := edgesForCycle(a.Graph, currencies)
		if !ok {
			continue
		}
		factor := cycleProfitFactor(path)
		if factor.LessThanOrEqual(thresholdDecimal) {
			continue
		}
		canon, rotated := canonicalCycle(currencies)
		key := strings.Join(canon, ">")
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		// Re-resolve path so it matches the canonical rotation, so
		// callers see Path[0].From == StartCurrency.
		canonPath, ok := edgesForCycle(a.Graph, rotated)
		if !ok {
			canonPath = path
		}
		out = append(out, ArbitrageCycle{
			Path:          canonPath,
			StartCurrency: canon[0],
			ProfitFactor:  factor,
			ProfitPercent: factor.Sub(decimal.NewFromInt(1)).Mul(decimal.NewFromInt(100)),
		})
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].ProfitFactor.GreaterThan(out[j].ProfitFactor)
	})
	return out, nil
}

// DetectFromCurrency returns only the cycles that pass through the given
// start currency. "Pass through" means the currency appears anywhere in the
// cycle — for an FX trader this is the practical reading, because every
// rotation of the cycle is a legal entry point.
//
// Example:
//
//	usdCycles, _ := d.DetectFromCurrency("USD")
func (a *ArbitrageDetector) DetectFromCurrency(start string) ([]ArbitrageCycle, error) {
	all, err := a.DetectCycles()
	if err != nil {
		return nil, err
	}
	if len(all) == 0 {
		return nil, nil
	}
	var out []ArbitrageCycle
	for _, c := range all {
		for _, edge := range c.Path {
			if edge.From == start {
				out = append(out, c)
				break
			}
		}
	}
	return out, nil
}

// --- internals ------------------------------------------------------------

type bfEdge struct {
	u, v   int
	weight float64
}

// effectiveRate is the per-edge rate after percentage fees, ignoring flat
// fees (see v1 simplification in the package comment). Returns
// decimal.Zero for any edge that would produce a non-positive effective
// rate — Bellman-Ford on log(0) or log(negative) is undefined.
func effectiveRate(r models.Rate) decimal.Decimal {
	if r.Value.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero
	}
	feeFactor := decimal.NewFromInt(1).Sub(r.FeePercentage)
	if feeFactor.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero
	}
	eff := r.Value.Mul(feeFactor)
	if eff.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero
	}
	return eff
}

// buildNodeIndex collects every currency referenced in the graph (both as
// edge source and edge target) and assigns each a stable integer index.
// Sorting makes Bellman-Ford's tie-breaking deterministic across runs.
func buildNodeIndex(g *models.Graph) ([]string, map[string]int) {
	seen := make(map[string]struct{})
	for src, edges := range g.Edges {
		seen[src] = struct{}{}
		for _, e := range edges {
			seen[e.To] = struct{}{}
		}
	}
	nodes := make([]string, 0, len(seen))
	for n := range seen {
		nodes = append(nodes, n)
	}
	sort.Strings(nodes)
	idx := make(map[string]int, len(nodes))
	for i, n := range nodes {
		idx[n] = i
	}
	return nodes, idx
}

// materializeEdges flattens g.Edges into a []bfEdge for the relaxation
// loop, skipping any edge whose effective rate is non-positive.
func materializeEdges(g *models.Graph, idx map[string]int) []bfEdge {
	var out []bfEdge
	for _, edges := range g.Edges {
		for _, r := range edges {
			eff := effectiveRate(r)
			if eff.LessThanOrEqual(decimal.Zero) {
				continue
			}
			out = append(out, bfEdge{
				u:      idx[r.From],
				v:      idx[r.To],
				weight: -math.Log(eff.InexactFloat64()),
			})
		}
	}
	return out
}

// walkCycle takes a node that relaxed on the V-th Bellman-Ford pass and
// returns the cycle it belongs to as a slice of node indices in traversal
// order, starting and ending at the same index.
//
// Walking predecessors n times guarantees we're inside the cycle (rather
// than on a tail leading into it); after that we walk until we see a
// node twice — the slice between the two sightings is the cycle.
func walkCycle(pred []int, start, n int) []int {
	cur := start
	for range n {
		if cur == -1 {
			return nil
		}
		cur = pred[cur]
	}
	if cur == -1 {
		return nil
	}

	seen := make(map[int]int)
	order := []int{}
	for cur != -1 {
		if pos, ok := seen[cur]; ok {
			cycle := append([]int{}, order[pos:]...)
			// reverse — predecessor walk goes backwards
			for i, j := 0, len(cycle)-1; i < j; i, j = i+1, j-1 {
				cycle[i], cycle[j] = cycle[j], cycle[i]
			}
			return cycle
		}
		seen[cur] = len(order)
		order = append(order, cur)
		cur = pred[cur]
	}
	return nil
}

// edgesForCycle resolves a sequence of currency codes back into the
// underlying Rate edges. When multiple providers quote the same corridor,
// the edge with the highest effective rate wins so the reported profit is
// the best achievable, not just the first one stored.
func edgesForCycle(g *models.Graph, cycle []string) ([]models.Rate, bool) {
	if len(cycle) < 2 {
		return nil, false
	}
	path := make([]models.Rate, 0, len(cycle))
	for i := range cycle {
		from := cycle[i]
		to := cycle[(i+1)%len(cycle)]
		edge, ok := bestEdge(g, from, to)
		if !ok {
			return nil, false
		}
		path = append(path, edge)
	}
	return path, true
}

// bestEdge returns the edge from→to with the highest effective rate.
func bestEdge(g *models.Graph, from, to string) (models.Rate, bool) {
	var best models.Rate
	bestRate := decimal.Zero
	found := false
	for _, e := range g.Edges[from] {
		if e.To != to {
			continue
		}
		eff := effectiveRate(e)
		if eff.GreaterThan(bestRate) {
			best = e
			bestRate = eff
			found = true
		}
	}
	return best, found
}

// cycleProfitFactor multiplies effective rates along the cycle in
// decimal.Decimal so the reported figure is exact (float64 is only used to
// drive Bellman-Ford internally).
func cycleProfitFactor(path []models.Rate) decimal.Decimal {
	product := decimal.NewFromInt(1)
	for _, e := range path {
		product = product.Mul(effectiveRate(e))
	}
	return product
}

// canonicalCycle rotates the cycle so its lexicographically smallest
// currency leads. The original (non-rotated) ordering would make
// {USD,EUR,GBP} and {EUR,GBP,USD} look like different cycles even though
// they describe the same opportunity. Returns the canonical key form and a
// matching rotated slice the caller can re-resolve edges from.
func canonicalCycle(currencies []string) (key, rotated []string) {
	if len(currencies) == 0 {
		return nil, nil
	}
	minIdx := 0
	for i, c := range currencies {
		if c < currencies[minIdx] {
			minIdx = i
		}
	}
	rotated = make([]string, len(currencies))
	for i := range currencies {
		rotated[i] = currencies[(minIdx+i)%len(currencies)]
	}
	key = rotated
	return key, rotated
}
