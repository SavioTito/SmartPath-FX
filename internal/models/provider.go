package models

type ExchangeProvider interface {
	FetchRates(base string) ([]Rate, error) //List of rates for a given base currency (e.g., "USD")
	Name() string                           //Name of the provider (Wise, Revolut, etc.)
}
