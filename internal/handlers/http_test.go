package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"commscope/internal/metrics"
	"commscope/internal/registry"
)

func TestHTTPHandler_SendAndStats(t *testing.T) {
	reg := registry.NewConnectionRegistry()
	tracker := metrics.NewProtocolTracker("http")
	handler := NewHTTPHandler(reg, tracker)

	// 1. Test POST /api/http/send
	reqBody := []byte(`{"client_id":"test-client","message":"hello"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/http/send", bytes.NewBuffer(reqBody))
	rec := httptest.NewRecorder()

	handler.HandleSend(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
	}

	// 2. Verify Registry and Metrics were updated
	if reg.Count() != 1 {
		t.Errorf("Expected 1 registered connection, got %d", reg.Count())
	}

	// 3. Test GET /api/http/stats
	statsReq := httptest.NewRequest(http.MethodGet, "/api/http/stats", nil)
	statsRec := httptest.NewRecorder()

	handler.HandleStats(statsRec, statsReq)

	if statsRec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, statsRec.Code)
	}

	var metricsSnapshot metrics.ProtocolMetrics
	if err := json.NewDecoder(statsRec.Body).Decode(&metricsSnapshot); err != nil {
		t.Fatalf("Failed to decode stats response: %v", err)
	}

	if metricsSnapshot.MessageCount != 1 {
		t.Errorf("Expected 1 total message in stats, got %d", metricsSnapshot.MessageCount)
	}
}
