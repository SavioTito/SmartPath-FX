package main

import (
	"fmt"

	"github.com/saviotito/currency-router/internal/models"
)

func main() {
	newGraph := models.NewGraph()
	firstRate := models.NewRate("USD", "EUR", 0.92, "Wise")
	secondRate := models.NewRate("EUR", "GBP", 0.85, "Wise")

	newGraph.AddRate(firstRate)
	newGraph.AddRate(secondRate)

	fmt.Printf("%+v\n", newGraph)
}
