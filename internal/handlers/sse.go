package handlers

import (
	"commscope/internal/metrics"
	"commscope/internal/registry"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type SSEBroadcaster struct {
	mu        sync.RWMutex
	listeners map[chan Message]struct{}
}

func NewSSEBroadcaster() *SSEBroadcaster {
	return &SSEBroadcaster{
		listeners: make(map[chan Message]struct{}),
	}

}

func (b *SSEBroadcaster) Subscribe() chan Message {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan Message, 10)
	b.listeners[ch] = struct{}{}
	return ch
}

func (b *SSEBroadcaster) Unsubscribe(ch chan Message) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.listeners, ch)
}

func (b *SSEBroadcaster) Broadcast(msg Message) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range b.listeners {
		select {
		case ch <- msg:
		default:
			// Non-blocking send: prevents a slow
			//  or disconnected listener from blocking other streams
		}
	}
}

type SSEHandler struct {
	registry    *registry.ConnectionRegistry
	tracker     *metrics.ProtocolTracker
	broadcaster *SSEBroadcaster
	store       *MessageStore
}

func NewSSEHandler(reg *registry.ConnectionRegistry, tracker *metrics.ProtocolTracker, store *MessageStore, broadcaster *SSEBroadcaster) *SSEHandler {
	return &SSEHandler{
		registry:    reg,
		tracker:     tracker,
		store:       store,
		broadcaster: broadcaster,
	}
}

// request DTO
type SSESENdMessageRequest struct {
	ClientID string `json:"client_id"`
	Text     string `json:"text"`
}

// response DTO
type SSESendMessageResponse struct {
	Status  string  `json:"status"`
	Message Message `json:"message"`
}

func (h *SSEHandler) HandleSend(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if r.Method != http.MethodPost {
		http.Error(w, "Method is not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SSESENdMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}
	if req.ClientID == "" {
		req.ClientID = "sse-client-default"
	}

	msg := Message{
		ID:        fmt.Sprintf("msg-%d", time.Now().UnixNano()),
		ClientID:  req.ClientID,
		Text:      req.Text,
		CreatedAt: start,
	}

	// Store Message
	h.store.Add(msg)

	//BroadCast live event to All active SSE streams!

	h.broadcaster.Broadcast(msg)
	//3. Track registry and telemetry

	h.registry.Add(&registry.Client{
		ID:          req.ClientID,
		Protocol:    "sse",
		ConnectedAt: start,
	})
	h.registry.RecordMessage(req.ClientID)
	duration := time.Since(start)
	h.tracker.Record(duration)

	//send JSON response
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(SSESendMessageResponse{
		Status:  "success",
		Message: msg,
	})

}

func (h *SSEHandler) HandleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	//type-assert http.Flusher to enable streaming
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming Unsupported", http.StatusInternalServerError)
		return
	}

	// set official SSE HTTP streaming headers

	//not a regular HTTP response but a long-lived connection that pushes events
	w.Header().Set("Content-Type", "text/event-stream")
	// SSE should not be cached, because it is a live data stream
	w.Header().Set("Cache-Control", "no-cache")
	//HTTP connection should remain open, generally not required for HHTP/2
	w.Header().Set("Connection", "keep-alive")
	//This allows browsers from other origins to access the SSE endpoint, * = allow request from any origin, for production put actual origin
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 3. Subscribe client channel to SSEBroadcaster

	//SSE client becomes a listener
	ch := h.broadcaster.Subscribe()
	defer h.broadcaster.Unsubscribe(ch)

	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		clientID = fmt.Sprintf("sse-listener-%d", time.Now().UnixNano())
	}

	// record connection time
	start := time.Now()

	h.registry.Add(&registry.Client{
		ID:          clientID,
		Protocol:    "sse",
		ConnectedAt: start,
	})

	defer h.registry.Remove(clientID)

	// 4. Stream events live until client disconnects
	for {
		select {
		case msg := <-ch:

			jsonBytes, err := json.Marshal(msg)
			if err != nil {
				continue
			}
			//Format standard SSE framing: data: <json>\n\n
			fmt.Fprintf(w, "data: %s\n\n", jsonBytes)
			flusher.Flush() // Flush chunk down tcp immediately

			h.registry.RecordMessage(clientID)
			h.tracker.Record(time.Since(start))
		case <-r.Context().Done():
			// Client closed browser tab stream
			return

		}
	}

}


func (h *SSEHandler) HandleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed",http.StatusMethodNotAllowed)
		return
	}
	activeConnections := h.registry.Count()
	clients := h.registry.List()

	var totalMessages int64
	for _, client := range clients{
		if client.Protocol == "sse" {
			totalMessages += client.MessageCount
		}
	}
	metricsSnapshot := h.tracker.Snapshot(activeConnections, totalMessages)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(metricsSnapshot)
}