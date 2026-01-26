package models

type CalculateRequest struct {
	From   string  `json:"from"`
	To     string  `json:"to"`
	Amount float64 `json:"amount"`
} // What we get from users

type CalculateResponse struct {
	Path        []Rate  `json:"path"`
	FinalAmount float64 `json:"final_amount"`
} //What we send back to them
