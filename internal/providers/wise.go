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
} // Name satisfies the ExchangeProvider interface.

//====== Implementing FetchRates ======

func (w *WiseProvider) FetchRates(base string) ([]models.Rate, error) {
	url := fmt.Sprintf("%s/v1/rates?source=%s", w.BaseURL, base)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+w.APIKey)
	req.Header.Set("User-Agent", "CurrencyRouter/1.0")

	fmt.Printf("DEBUG: Sending request to %s...\n", url)
	resp, err := w.Client.Do(req)
	if err != nil {
		fmt.Printf("Wise API Status: %d for base %s\n", resp.StatusCode, base)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wise api returned status: %d", resp.StatusCode)
	}

	var rawRates []WiseResponse
	if err := json.NewDecoder(resp.Body).Decode(&rawRates); err != nil {
		return nil, err
	}

	var rates []models.Rate
	for _, r := range rawRates {
		rates = append(rates, models.NewRate(r.Source, r.Target, r.Rate, "Wise"))
	} // Convert WiseResponse (External) to models.Rate (Internal)

	return rates, nil
}
