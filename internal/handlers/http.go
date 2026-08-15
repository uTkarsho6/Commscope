package handlers

import (
	"commscope/internal/metrics"
	"commscope/internal/registry"
	"encoding/json"
	"net/http"
	"time"
)

type HTTPHandler struct {
	registry *registry.ConnectionRegistry
	tracker  *metrics.ProtocolTracker
}

func NewHTTPHandler(reg *registry.ConnectionRegistry, tracker *metrics.ProtocolTracker) *HTTPHandler {
	return &HTTPHandler{
		registry: reg,
		tracker:  tracker,
	}
}

type sendMessageRequest struct {
	ClientID string `json:"client_id"`
	Message  string `json:"message"`
}

type SendMessageResponse struct {
	Status    string  `json:"status"`
	ClientID  string  `json:"client_id"`
	Message   string  `json:"message"`
	LatencyMs float64 `json:"latency_ms"`
}

func (h *HTTPHandler) HandleSend(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if r.Method != http.MethodPost {
		http.Error(w, "Method Not allowed", http.StatusMethodNotAllowed) //405 method not allowed
		return
	}

	var req sendMessageRequest

	//decode the request body

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest) // 400 bad request
		return
	}

	if req.ClientID == "" {
		req.ClientID = "http-client-default"
	}

	h.registry.Add(&registry.Client{
		ID:          req.ClientID,
		Protocol:    "http",
		ConnectedAt: start,
	})
	h.registry.RecordMessage(req.ClientID)

	duration := time.Since(start)
	h.tracker.Record(duration)
	w.Header().Set("Content-Type", "application/json") // telling client response will be in json format
	w.WriteHeader(http.StatusOK)                       // The request was successfully processed.

	json.NewEncoder(w).Encode(SendMessageResponse{
		Status:    "success",
		ClientID:  req.ClientID,
		Message:   req.Message,
		LatencyMs: float64(duration.Microseconds()) / 1000.0,
	})

}

func (h *HTTPHandler) HandleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet { //check for GET /stats
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	activeConnections := h.registry.Count()
	clients := h.registry.List()

	var totalMessages int64

	for _, client := range clients {
		totalMessages += client.MessageCount
	}
	metricsSnapShot := h.tracker.Snapshot(activeConnections, totalMessages)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(metricsSnapShot)

}
