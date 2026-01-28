package providers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/saviotito/currency-router/internal/models"
)

type WiseResponse struct {
	Source string  `json:"source"`
	Target string  `json:"target"`
	Rate   float64 `json:"rate"`
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
		Source string  `json:"source"`
		Target string  `json:"target"`
		Value  float64 `json:"rate"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rawRates); err != nil {
		return nil, err
	}

	var rates []models.Rate
	for _, r := range rawRates {
		fee, feeCur := w.EstimateFee(r.Source, r.Target, 1000)

		rates = append(rates, models.Rate{
			From:        r.Source,
			To:          r.Target,
			Value:       r.Value,
			FixedFee:    fee,
			FeeCurrency: feeCur,
			Provider:    w.Name(),
			LastUpdate:  time.Now(),
		})
	}
	return rates, nil
}

func (w *WiseProvider) EstimateFee(source, target string, amount float64) (float64, string) {
	basePercentage := 0.004
	flatFee := 2.00

	if amount > 100000 {
		flatFee = flatFee * 0.8 // 20% discount on fixed portion
	}

	totalFee := flatFee + (amount * basePercentage)

	return totalFee, source
}
