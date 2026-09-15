package webrtcserver

import (
	"bytes"
	"commscope/internal/metrics"
	"commscope/internal/registry"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWebRTCHandler_HandleOffer_InvalidMethod(t *testing.T) {
	reg := registry.NewConnectionRegistry()
	tracker := metrics.NewProtocolTracker("webrtc")
	handler := NewWebRTCHandler(reg, tracker)

	req := httptest.NewRequest(http.MethodGet, "/api/webrtc/offer", nil)
	rec := httptest.NewRecorder()

	handler.HandleOffer(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status code %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

func TestWebRTCHandler_HandleOffer_InvalidJSON(t *testing.T) {
	reg := registry.NewConnectionRegistry()
	tracker := metrics.NewProtocolTracker("webrtc")
	handler := NewWebRTCHandler(reg, tracker)

	req := httptest.NewRequest(http.MethodPost, "/api/webrtc/offer", bytes.NewBufferString("invalid json"))
	rec := httptest.NewRecorder()

	handler.HandleOffer(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestWebRTCHandler_HandleStats(t *testing.T) {
	reg := registry.NewConnectionRegistry()
	tracker := metrics.NewProtocolTracker("webrtc")
	handler := NewWebRTCHandler(reg, tracker)

	req := httptest.NewRequest(http.MethodGet, "/api/webrtc/stats", nil)
	rec := httptest.NewRecorder()

	handler.HandleStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
	}

	var snap metrics.ProtocolMetrics
	if err := json.NewDecoder(rec.Body).Decode(&snap); err != nil {
		t.Fatalf("Failed to decode JSON stats response: %v", err)
	}

	if snap.Protocol != "webrtc" {
		t.Errorf("Expected protocol 'webrtc', got '%s'", snap.Protocol)
	}
}

