package api

import (
	"encoding/json"
	"net/http"
	"testing"
)

// TestHealthCheck tests the health check endpoint
func TestHealthCheck(t *testing.T) {
	handler := &Handler{}
	c, rec := newTestContext(http.MethodGet, "/healthz", "")

	err := handler.HealthCheck(c)
	if err != nil {
		t.Fatalf("HealthCheck() error = %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["status"] != "healthy" {
		t.Errorf("expected status 'healthy', got '%s'", resp["status"])
	}

	if resp["time"] == "" {
		t.Error("expected non-empty time field")
	}
}
