package models

type CalculateRequest struct {
	From string `json:"from"`
	To   string `json:"to"`
} // What we get from users

type CalculateResponse struct {
	Path  []Rate  `json:"path"`
	Total float64 `json:"total_rate"`
} //What we send back to them
