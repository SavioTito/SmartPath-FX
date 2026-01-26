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
		// In a real production environment, you might hit /v1/prices for top pairs.
		fee := w.estimateFixedFee(r.Source, r.Target)
		rates = append(rates, models.NewRate(r.Source, r.Target, r.Value, fee, w.Name()))
	}
	return rates, nil
}

func (w *WiseProvider) estimateFixedFee(source, target string) float64 {
	switch source {
	case "USD":
		return 4.00
	case "EUR":
		return 0.50
	case "GBP":
		return 0.30
	default:
		return 2.00
	}
}
