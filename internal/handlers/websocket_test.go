package handlers

import (
	"bytes"
	"commscope/internal/metrics"
	"commscope/internal/registry"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestWebSocketHandler_ConnectSendStats(t *testing.T) {
	reg := registry.NewConnectionRegistry()
	tracker := metrics.NewProtocolTracker("ws")
	hub := NewWSHub()
	go hub.Run()

	handler := NewWSHandler(hub, reg, tracker)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/ws/connect", handler.HandleConnect)
	mux.HandleFunc("/api/ws/send", handler.HandleSend)
	mux.HandleFunc("/api/ws/stats", handler.HandleStats)

	server := httptest.NewServer(mux)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/ws/connect?client_id=test-ws-client"

	//connect WebSocket client

	wsConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)

	if err != nil {
		t.Fatalf("failed to connect to Websocket client: %v", err)

	}
	defer wsConn.Close()

	//send msg via POST endpoint
	sendpayLoad := map[string]string{
		"client_id": "test-ws-client",
		"text":      "Hello Websocket",
	}
	//Marshal returns the JSON encoding of v.
	body, _ := json.Marshal(sendpayLoad)

	resp, err := http.Post(server.URL+"/api/ws/send", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to POST /api/ws/send %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	//   Read message from WebSocket connection
	wsConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msgBytes, err := wsConn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read message form websocket: %v", err)
	}

	if !strings.Contains(string(msgBytes), "Hello Websocket") {
		t.Errorf("Expected message to contain 'Hello websocket', got %s", string(msgBytes))
	}

	// check stats
	statsResp, err := http.Get(server.URL + "/api/ws/stats")
	if err != nil {
		t.Fatalf("Failed to GET /api/ws/stats %v", err)
	}

	var stats metrics.ProtocolMetrics
	if err := json.NewDecoder(statsResp.Body).Decode(&stats); err != nil {
		t.Fatalf("Failed to decode stats %v", err)
	}

	if stats.ActiveConnections != 1 {
		t.Errorf("Expected 1 active connection, got %d", stats.ActiveConnections)

	}

}
