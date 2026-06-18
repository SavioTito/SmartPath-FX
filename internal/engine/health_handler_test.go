package engine

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/saviotito/currency-router/internal/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type noopProvider struct{}

func (n *noopProvider) Name() string { return "Mock" }
func (n *noopProvider) QuoteFee(_, _ string, _ decimal.Decimal) (models.FeeQuote, error) {
	return models.FeeQuote{}, nil
}
func (n *noopProvider) QuoteAllProviders(_, _ string) (map[string]models.FeeQuote, error) {
	return nil, nil
}

func TestHealth_ReturnsOk(t *testing.T) {
	h := &HealthHandler{
		StartTime: time.Now(),
		Providers: []models.ExchangeProvider{&noopProvider{}},
		Cache:     NewMemoryCache(),
	}
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp healthResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "ok", resp.Status)
	assert.GreaterOrEqual(t, resp.UptimeSeconds, int64(0))
	assert.Equal(t, "ok", resp.Checks["graph_cache"])
	assert.Equal(t, "ok", resp.Checks["mock_provider"])
}
