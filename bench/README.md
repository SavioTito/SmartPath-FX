# Benchmarks

All benchmarks run in-memory (no network). Graphs are synthetic but
structurally realistic. Measured on Intel Core i9-9980HK @ 2.40 GHz,
Go 1.25.5, darwin/amd64.

## What each benchmark measures

### BenchmarkFindBestRoute

Dijkstra-based routing across a fully synthetic DAG. Sub-benchmarks
cover 10, 50, and 100 currencies with ~5 outgoing edges per node.
Each iteration calls `Router.FindBestRoute("C000", "C099", 1000)`.
Graph construction is excluded from timing via `b.ResetTimer()`.

### BenchmarkDetectCycles

Bellman-Ford negative-cycle detection. Graph has a ring backbone plus
cross edges (~1.5× ring density) and one injected profitable triangle
(C000→C001→C002→C000, product 1.331). Sub-benchmarks cover 10, 30,
and 100 currencies. Each iteration calls `ArbitrageDetector.DetectCycles()`.

### BenchmarkAggregatorMerge

In-memory graph assembly: 50 `Rate` objects spanning 5 providers and
10 real-world corridors (USD/EUR, GBP/JPY, etc.). Each iteration calls
`models.NewGraph()` followed by 50 `AddRate` calls — mirroring the
hot path inside `Aggregator.FetchSmartGraph` without the HTTP layer.

## How to reproduce

```bash
go test -bench=. -benchmem -benchtime=2s -count=3 ./internal/engine/ \
  | tee bench/benchmarks.txt

go install golang.org/x/perf/cmd/benchstat@latest
benchstat bench/benchmarks.txt > bench/benchmarks_summary.txt
```

benchstat needs ≥ 6 samples (i.e., `-count=6`) for 95% confidence
intervals. The files here used `-count=3` as a baseline.
