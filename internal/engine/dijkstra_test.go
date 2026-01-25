package engine

import (
	"testing"

	"github.com/saviotito/currency-router/internal/models"
)

func TestFindBestRoute(t *testing.T) {
	// 1. Setup the "Mock" Data
	g := models.NewGraph()

	// Route A: USD -> EUR -> AOA (Total Rate: 0.9 * 1000 = 900)
	g.AddRate(models.NewRate("USD", "EUR", 0.90, "Wise"))
	g.AddRate(models.NewRate("EUR", "AOA", 1000, "Wise"))

	// Route B: USD -> GBP -> AOA (Total Rate: 1.1 * 1200 = 1320)
	// This should be the winner!
	g.AddRate(models.NewRate("USD", "GBP", 1.1, "Wise"))
	g.AddRate(models.NewRate("GBP", "AOA", 1200, "Wise"))

	t.Run("Finds the highest return path(lowest log weight)", func(t *testing.T) {
		path, err := FindBestRoute(g, "USD", "AOA")

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// We expect 2 steps: USD -> GBP and GBP -> AOA
		if len(path) != 2 {
			t.Errorf("Expected path length 2, got %d", len(path))
		}

		// Verify it chose the GBP route (the better rate)
		if path[0].To != "GBP" {
			t.Errorf("Wrong path! Expected USD -> GBP, but got USD -> %s", path[0].To)
		}
	})

	t.Run("Returns error for impossible route", func(t *testing.T) {
		_, err := FindBestRoute(g, "USD", "JPY")
		if err == nil {
			t.Error("Expected error for non-existent route, but got nil")
		}
	})
}
