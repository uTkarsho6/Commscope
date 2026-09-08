package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"commscope/internal/metrics"
	"commscope/internal/registry"
)
func TestSSEHandler_SendEventsStats(t *testing.T) {
	reg := registry.NewConnectionRegistry()
	tracker := metrics.NewProtocolTracker("sse")
	store := NewMessageStore()
	broadcaster := NewSSEBroadcaster()
	handler := NewSSEHandler(reg, tracker, store, broadcaster)
	// 1. Test POST /api/sse/send
	reqBody := []byte(`{"client_id":"sse-sender","text":"initial event"}`)
	sendReq := httptest.NewRequest(http.MethodPost, "/api/sse/send", bytes.NewBuffer(reqBody))
	sendRec := httptest.NewRecorder()
	handler.HandleSend(sendRec, sendReq)
	if sendRec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, sendRec.Code)
	}
	// 2. Test GET /api/sse/events with stream cancellation
	ctx, cancel := context.WithCancel(context.Background())
	eventsReq := httptest.NewRequest(http.MethodGet, "/api/sse/events?client_id=sse-listener", nil).WithContext(ctx)
	eventsRec := httptest.NewRecorder()
	// Launch stream handler in a background goroutine
	done := make(chan struct{})
	go func() {
		handler.HandleEvents(eventsRec, eventsReq)
		close(done)
	}()
	// Wait 20ms for subscription, then broadcast an event
	time.Sleep(20 * time.Millisecond)
	postBody := []byte(`{"client_id":"sse-sender","text":"live event chunk"}`)
	postReq := httptest.NewRequest(http.MethodPost, "/api/sse/send", bytes.NewBuffer(postBody))
	postRec := httptest.NewRecorder()
	handler.HandleSend(postRec, postReq)
	// Allow event to flush, then cancel the stream
	time.Sleep(20 * time.Millisecond)
	cancel()
	<-done
	// Verify headers and response stream framing
	contentType := eventsRec.Header().Get("Content-Type")
	if contentType != "text/event-stream" {
		t.Errorf("Expected Content-Type 'text/event-stream', got '%s'", contentType)
	}
	bodyStr := eventsRec.Body.String()
	if !strings.Contains(bodyStr, "data:") || !strings.Contains(bodyStr, "live event chunk") {
		t.Errorf("Expected streamed data chunk in body, got '%s'", bodyStr)
	}
	// 3. Test GET /api/sse/stats
	statsReq := httptest.NewRequest(http.MethodGet, "/api/sse/stats", nil)
	statsRec := httptest.NewRecorder()
	handler.HandleStats(statsRec, statsReq)
	if statsRec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, statsRec.Code)
	}
}
