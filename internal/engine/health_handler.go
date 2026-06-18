package engine

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/saviotito/currency-router/internal/models"
	"github.com/saviotito/currency-router/internal/version"
)

// HealthHandler handles GET /healthz.
type HealthHandler struct {
	StartTime time.Time
	Providers []models.ExchangeProvider
	Cache     *MemoryChace
}

type healthResponse struct {
	Status        string            `json:"status"`
	Version       string            `json:"version"`
	UptimeSeconds int64             `json:"uptime_seconds"`
	Checks        map[string]string `json:"checks"`
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	checks := make(map[string]string)
	allOk := true

	if h.Cache != nil {
		checks["graph_cache"] = "ok"
	} else {
		checks["graph_cache"] = "fail"
		allOk = false
	}

	for _, p := range h.Providers {
		key := strings.ToLower(p.Name()) + "_provider"
		if p.Name() != "" {
			checks[key] = "ok"
		} else {
			checks[key] = "fail"
			allOk = false
		}
	}

	status := "ok"
	code := http.StatusOK
	if !allOk {
		status = "degraded"
		code = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(healthResponse{
		Status:        status,
		Version:       version.Version,
		UptimeSeconds: int64(time.Since(h.StartTime).Seconds()),
		Checks:        checks,
	})
}
