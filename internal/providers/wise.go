package providers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/saviotito/currency-router/internal/models"
	"github.com/shopspring/decimal"
)

type WiseResponse struct {
	Source string          `json:"source"`
	Target string          `json:"target"`
	Rate   decimal.Decimal `json:"rate"`
}

type WiseProvider struct {
	Client  *http.Client
	APIKey  string
	BaseURL string
}

func NewWiseProvider(apiKey, baseURL string) *WiseProvider {
	return &WiseProvider{
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
		APIKey:  apiKey,
		BaseURL: baseURL,
	}
}

func (w *WiseProvider) Name() string {
	return "Wise"
}

//====== Implementing FetchRates ======

func (w *WiseProvider) FetchRates(base string) ([]models.Rate, error) {
	url := fmt.Sprintf("%s/v1/rates?source=%s", w.BaseURL, base)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+w.APIKey)

	resp, err := w.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var rawRates []struct {
		Source string      `json:"source"`
		Target string      `json:"target"`
		Value  json.Number `json:"rate"`
	}

	dec := json.NewDecoder(resp.Body)
	dec.UseNumber()
	if err := dec.Decode(&rawRates); err != nil {
		return nil, err
	}

	estimationAmount := decimal.NewFromInt(1000)

	var rates []models.Rate
	for _, r := range rawRates {
		value, err := decimal.NewFromString(r.Value.String())
		if err != nil {
			fmt.Printf("WARN: skipping %s->%s, bad rate %q: %v\n", r.Source, r.Target, r.Value, err)
			continue
		}

		fee, feeCur := w.EstimateFee(r.Source, r.Target, estimationAmount)

		rates = append(rates, models.Rate{
			From:        r.Source,
			To:          r.Target,
			Value:       value,
			FixedFee:    fee,
			FeeCurrency: feeCur,
			Provider:    w.Name(),
			LastUpdate:  time.Now(),
		})
	}
	return rates, nil
}

func (w *WiseProvider) EstimateFee(source, target string, amount decimal.Decimal) (decimal.Decimal, string) {
	basePercentage := decimal.NewFromFloat(0.004)
	flatFee := decimal.NewFromFloat(2.00)

	if amount.GreaterThan(decimal.NewFromInt(100000)) {
		flatFee = flatFee.Mul(decimal.NewFromFloat(0.8)) // 20% discount on fixed portion
	}

	totalFee := flatFee.Add(amount.Mul(basePercentage))

	return totalFee, source
}
