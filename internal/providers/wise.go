package providers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/saviotito/currency-router/internal/models"
)

type WiseResponse struct {
	Source string  `json:"source"`
	Target string  `json:"target"`
	Value  float64 `json:"value"`
}

type WiseProvider struct {
	Client *http.Client
	APIKey string
}

func NewWiseProvider(apiKey string) *WiseProvider {
	return &WiseProvider{
		Client: &http.Client{},
		APIKey: apiKey,
	}
}

func (w *WiseProvider) Name() string {
	return "Wise"
} // Name satisfies the ExchangeProvider interface.

//====== Implementing FetchRates ======

func (w *WiseProvider) FetchRates(base string) ([]models.Rate, error) {
	url := fmt.Sprintf("https://api.wise.com/v1/rates?source=%s", base)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer"+w.APIKey)

	resp, err := w.Client.Do(req)
	if err != nil {
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
		rates = append(rates, models.NewRate(r.Source, r.Target, r.Value, "Wise"))
	} // Convert WiseResponse (External) to models.Rate (Internal)

	return rates, nil
}
