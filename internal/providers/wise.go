package providers

// WiseProvider is misnamed historically — it actually drives Wise's public
// /v4/comparisons/ endpoint, which returns quotes for EVERY major remittance
// provider (Wise, Western Union, XE, Remitly, Xoom, OFX, banks). Rename to
// ComparisonProvider when a sweep PR can touch main.go safely.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/saviotito/currency-router/internal/models"
	"github.com/shopspring/decimal"
)

const wiseComparisonBaseURL = "https://api.wise.com"

var (
	probeLow  = decimal.NewFromInt(100)
	probeHigh = decimal.NewFromInt(10000)
)

var comparisonSem = make(chan struct{}, 6)

const (
	probeMaxAttempts = 3
	probeBackoffBase = 400 * time.Millisecond
	// Drop providers whose implied fee on the high probe exceeds 5% — banks
	// and partner rails that aren't realistic competitors clutter the graph
	// without ever being picked.
	maxRealisticFeeRatio = 0.05
)

type WiseProvider struct {
	Client *http.Client

	quoteCacheMu  sync.RWMutex
	quoteCache    map[string]cachedQuoteMap
	quoteCacheTTL time.Duration
}

type cachedQuoteMap struct {
	quotes map[string]models.FeeQuote
	at     time.Time
}

func NewWiseProvider() *WiseProvider {
	return &WiseProvider{
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
		quoteCache:    make(map[string]cachedQuoteMap),
		quoteCacheTTL: 5 * time.Minute,
	}
}

func (w *WiseProvider) Name() string {
	return "Wise"
}

// QuoteFee returns the Wise-only quote, satisfying callers that don't care
// about competitors. Thin filter over QuoteAllProviders.
func (w *WiseProvider) QuoteFee(source, target string, _ decimal.Decimal) (models.FeeQuote, error) {
	quotes, err := w.QuoteAllProviders(source, target)
	if err != nil {
		return models.FeeQuote{}, err
	}
	if q, ok := quotes["wise"]; ok {
		return q, nil
	}
	return models.FeeQuote{}, fmt.Errorf("wise quote missing for %s->%s", source, target)
}

// QuoteAllProviders returns one FeeQuote per provider that quoted the
// corridor with a linear (flat + percentage) schedule. Map key is the
// provider's lowercase alias.
func (w *WiseProvider) QuoteAllProviders(source, target string) (map[string]models.FeeQuote, error) {
	cacheKey := fmt.Sprintf("%s|%s", source, target)
	w.quoteCacheMu.RLock()
	if cached, ok := w.quoteCache[cacheKey]; ok && time.Since(cached.at) < w.quoteCacheTTL {
		w.quoteCacheMu.RUnlock()
		out := make(map[string]models.FeeQuote, len(cached.quotes))
		for k, v := range cached.quotes {
			v.Source = "cached"
			out[k] = v
		}
		return out, nil
	}
	w.quoteCacheMu.RUnlock()

	type probeResult struct {
		quotes []providerQuote
		err    error
	}
	low := make(chan probeResult, 1)
	high := make(chan probeResult, 1)
	go func() {
		q, err := w.probeComparison(source, target, probeLow)
		low <- probeResult{q, err}
	}()
	go func() {
		q, err := w.probeComparison(source, target, probeHigh)
		high <- probeResult{q, err}
	}()

	lo, hi := <-low, <-high
	if lo.err != nil {
		return nil, lo.err
	}
	if hi.err != nil {
		return nil, hi.err
	}

	loByAlias := indexByAlias(lo.quotes)
	hiByAlias := indexByAlias(hi.quotes)
	spread := probeHigh.Sub(probeLow)
	out := make(map[string]models.FeeQuote)
	now := time.Now()

	for alias, hiq := range hiByAlias {
		loq, ok := loByAlias[alias]
		if !ok {
			continue // present in only one probe; non-linear or flaky
		}

		percentage := hiq.fee.Sub(loq.fee).Div(spread)
		flat := loq.fee.Sub(percentage.Mul(probeLow))

		// Reject non-linear schedules (fees that drop with size, like Xoom's
		// "free above $X" tier). Negative pct or negative flat means the
		// linear model can't represent this provider safely.
		if percentage.LessThan(decimal.Zero) || flat.LessThan(decimal.Zero) {
			continue
		}

		// Cull unrealistically expensive providers (banks, partner rails).
		if !hiq.fee.IsZero() && hiq.fee.Div(probeHigh).GreaterThan(decimal.NewFromFloat(maxRealisticFeeRatio)) {
			continue
		}

		out[alias] = models.FeeQuote{
			Provider:   hiq.name,
			Rate:       hiq.rate,
			Flat:       flat,
			Percentage: percentage,
			Currency:   source,
			Source:     "live",
			FetchedAt:  now,
		}
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("no linear-fit providers for %s->%s", source, target)
	}

	w.quoteCacheMu.Lock()
	stored := make(map[string]models.FeeQuote, len(out))
	for k, v := range out {
		stored[k] = v
	}
	w.quoteCache[cacheKey] = cachedQuoteMap{quotes: stored, at: time.Now()}
	w.quoteCacheMu.Unlock()

	return out, nil
}

type providerQuote struct {
	alias, name string
	fee, rate   decimal.Decimal
}

func indexByAlias(qs []providerQuote) map[string]providerQuote {
	out := make(map[string]providerQuote, len(qs))
	for _, q := range qs {
		out[strings.ToLower(q.alias)] = q
	}
	return out
}

// probeComparison hits the public comparison endpoint and returns every
// provider's (alias, name, fee, rate) for the given sendAmount. Fee is in
// sourceCurrency. Throttled via comparisonSem; retries 429 with jittered
// backoff.
func (w *WiseProvider) probeComparison(source, target string, sendAmount decimal.Decimal) ([]providerQuote, error) {
	// Wise /v4/comparisons/ is a public endpoint — no API key required.
	comparisonSem <- struct{}{}
	defer func() { <-comparisonSem }()

	q := url.Values{}
	q.Set("sourceCurrency", source)
	q.Set("targetCurrency", target)
	q.Set("sendAmount", sendAmount.String())
	endpoint := fmt.Sprintf("%s/v4/comparisons/?%s", wiseComparisonBaseURL, q.Encode())

	var lastErr error
	for attempt := 0; attempt < probeMaxAttempts; attempt++ {
		if attempt > 0 {
			delay := probeBackoffBase * time.Duration(1<<attempt)
			jitter := time.Duration(len(source+target)) * 17 * time.Millisecond
			time.Sleep(delay + jitter)
		}

		req, _ := http.NewRequest("GET", endpoint, nil)
		resp, err := w.Client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			lastErr = fmt.Errorf("wise comparison 429")
			continue
		}
		if resp.StatusCode >= 400 {
			resp.Body.Close()
			return nil, fmt.Errorf("wise comparison %d", resp.StatusCode)
		}

		var parsed struct {
			Providers []struct {
				Alias  string `json:"alias"`
				Name   string `json:"name"`
				Quotes []struct {
					Fee  decimal.Decimal `json:"fee"`
					Rate decimal.Decimal `json:"rate"`
				} `json:"quotes"`
			} `json:"providers"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()

		out := make([]providerQuote, 0, len(parsed.Providers))
		for _, p := range parsed.Providers {
			if len(p.Quotes) == 0 {
				continue
			}
			out = append(out, providerQuote{
				alias: p.Alias,
				name:  p.Name,
				fee:   p.Quotes[0].Fee,
				rate:  p.Quotes[0].Rate,
			})
		}
		return out, nil
	}
	return nil, lastErr
}

// FallbackQuote returns Wise's published typical schedule when the live
// comparison API is unreachable. Caller-side fallback path; not used in
// the normal multi-provider flow.
func FallbackQuote(source string) models.FeeQuote {
	return models.FeeQuote{
		Provider:   "Wise",
		Flat:       decimal.NewFromFloat(2.00),
		Percentage: decimal.NewFromFloat(0.004),
		Currency:   source,
		Source:     "fallback",
		FetchedAt:  time.Now(),
	}
}
