package engine

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/saviotito/currency-router/internal/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubFetcher struct{ g *models.Graph }

func (s *stubFetcher) FetchFullGraph(_ []string) *models.Graph { return s.g }

func TestArbitrageScan_HappyPath(t *testing.T) {
	g := buildGraph(
		makeRate("USD", "EUR", 0.9, 0),
		makeRate("EUR", "GBP", 0.8, 0),
		makeRate("GBP", "USD", 1.05/(0.9*0.8), 0),
	)
	h := &ArbitrageScanHandler{Fetcher: &stubFetcher{g}, Cache: NewMemoryCache()}
	req := httptest.NewRequest(http.MethodPost, "/arbitrage/scan", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp arbitrageScanResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Len(t, resp.Cycles, 1)

	expected := decimal.NewFromFloat(1.05)
	diff := resp.Cycles[0].ProfitFactor.Sub(expected).Abs()
	assert.True(t, diff.LessThan(decimal.NewFromFloat(0.001)),
		"profit_factor %s not within 0.001 of 1.05", resp.Cycles[0].ProfitFactor)
	assert.Equal(t, "live", resp.GraphSource)
}

func TestArbitrageScan_NoOpportunities(t *testing.T) {
	g := buildGraph(
		makeRate("USD", "EUR", 0.9, 0),
		makeRate("EUR", "GBP", 0.8, 0),
	)
	h := &ArbitrageScanHandler{Fetcher: &stubFetcher{g}, Cache: NewMemoryCache()}
	req := httptest.NewRequest(http.MethodPost, "/arbitrage/scan", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp arbitrageScanResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	// Must be [] not null.
	assert.NotNil(t, resp.Cycles)
	assert.Empty(t, resp.Cycles)
}

func TestArbitrageFrom_FiltersCorrectly(t *testing.T) {
	g := buildGraph(
		// USD-touching cycle
		makeRate("USD", "EUR", 0.9, 0),
		makeRate("EUR", "GBP", 0.8, 0),
		makeRate("GBP", "USD", 1.05/(0.9*0.8), 0),
		// JPY-only cycle (no USD)
		makeRate("JPY", "CAD", 0.01, 0),
		makeRate("CAD", "AUD", 1.2, 0),
		makeRate("AUD", "JPY", 1.03/(0.01*1.2), 0),
	)
	h := &ArbitrageFromHandler{Fetcher: &stubFetcher{g}, Cache: NewMemoryCache()}

	mux := http.NewServeMux()
	mux.Handle("GET /arbitrage/from/{currency}", h)

	req := httptest.NewRequest(http.MethodGet, "/arbitrage/from/USD", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp arbitrageFromResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "USD", resp.StartCurrency)
	assert.Len(t, resp.Cycles, 1)
	assert.True(t, containsCurrency(resp.Cycles[0].Path, "USD"))
}
