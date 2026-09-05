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

func TestPollingHandler_SendGetStats(t *testing.T) {
	reg := registry.NewConnectionRegistry()
	tracker := metrics.NewProtocolTracker("polling")
	store := NewMessageStore()
	handler := NewPollingHandler(reg, tracker, store)

	// Testing for POST "/api/polling/send"

	reqBody := []byte(`{"client_id" : "poller-1", "text": "hello short polling"}`)

	sendReq := httptest.NewRequest(http.MethodPost, "/api/polling/send", bytes.NewBuffer(reqBody))

	sendRec := httptest.NewRecorder()

	handler.HandleSend(sendRec, sendReq)
	if sendRec.Code != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, sendRec.Code)

	}

	// Testing GET "/api/polling/messages"

	getReq := httptest.NewRequest(http.MethodGet, "/api.polling/messages?since=0&client_id=poller-1", nil)

	getRec := httptest.NewRecorder()

	handler.HandleGetMessage(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, getRec.Code)
	}

	var getResp PollGetMessagesResponse
	if err := json.NewDecoder(getRec.Body).Decode(&getResp); err != nil {
		t.Fatalf("Failed to Decode poll response %v", err)
	}

	if getResp.Messages[0].Text != "hello short polling" {
		t.Errorf("Expected message was hellpo short polling but received '%s'", getResp.Messages[0].Text)

	}

	// testing of /api/polling/stats

	statsReq := httptest.NewRequest(http.MethodGet, "/api/polling/stats", nil)
	statsRec := httptest.NewRecorder()

	handler.HandleStats(statsRec, statsReq)

	if statsRec.Code != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, statsRec.Code)
	}

}
