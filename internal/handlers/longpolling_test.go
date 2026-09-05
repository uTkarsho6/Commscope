package handlers

import (
	"bytes"
	"commscope/internal/metrics"
	"commscope/internal/registry"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestLongPollingHandler_SendGetStats(t *testing.T) {
	reg := registry.NewConnectionRegistry()
	tracker := metrics.NewProtocolTracker("longpolling")
	store := NewMessageStore()
	broadcaster := NewLongPollBroadcaster()
	handler := NewLongPollingHandler(reg, tracker, store, broadcaster)

	// Test POST /api/longpolling/send

	reqBody := []byte(`{"client_id":"lp-client-1","text":"hello long polling"}`)
	sendReq := httptest.NewRequest(http.MethodPost, "/api/longpolling/send", bytes.NewBuffer(reqBody))
	sendRec := httptest.NewRecorder()

	handler.HandleSend(sendRec, sendReq)
	if sendRec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, sendRec.Code)
	}

	// 2. Test GET /api/longpolling/messages?since=0 (immediate match)
	getReq := httptest.NewRequest(http.MethodGet, "/api/longpolling/messages?since=0&client_id=lp-client-1", nil)
	getRec := httptest.NewRecorder()
	handler.HandleGetMessages(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, getRec.Code)
	}
	var getResp LongPollGetMessagesResponse
	if err := json.NewDecoder(getRec.Body).Decode(&getResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if getResp.Count != 1 || getResp.Messages[0].Text != "hello long polling" {
		t.Errorf("Unexpected message response: %+v", getResp)
	}

	// 3. Test Hanging GET with background broadcast!

	// Start hanging request in background (no response written yet)
	time.Sleep(10 * time.Millisecond)
	nowStr := strconv.FormatInt(time.Now().UnixMilli(), 10)

	// Launch background goroutine that posts a message after 50ms
	go func() {
		time.Sleep(50 * time.Millisecond)
		postBody := []byte(`{"client_id":"lp-client-2","text":"async arrival"}`)
		postReq := httptest.NewRequest(http.MethodPost, "/api/longpolling/send", bytes.NewBuffer(postBody))
		postRec := httptest.NewRecorder()
		handler.HandleSend(postRec, postReq)
	}()

	// Main test hangs on GET until background broadcast fires

	hangReq := httptest.NewRequest(http.MethodGet, "/api/longpolling/messages?since="+nowStr+"&timeout=2&client_id=lp-client-2", nil)
	hangRec := httptest.NewRecorder()
	handler.HandleGetMessages(hangRec, hangReq)

	if hangRec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, hangRec.Code)
	}

	var hangResp LongPollGetMessagesResponse
	json.NewDecoder(hangRec.Body).Decode(&hangResp)
	if hangResp.Count != 1 || hangResp.Messages[0].Text != "async arrival" {
		t.Errorf("Expected async message 'async arrival', got %+v", hangResp)
	}

	// 4. Test GET /api/longpolling/stats

	statsReq := httptest.NewRequest(http.MethodGet, "/api/longpolling/stats", nil)
	statsRec := httptest.NewRecorder()
	handler.HandleStats(statsRec, statsReq)
	if statsRec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, statsRec.Code)
	}
}
