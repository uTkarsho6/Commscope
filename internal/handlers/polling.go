package handlers

import (
	"commscope/internal/metrics"
	"commscope/internal/registry"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type Message struct {
	ID        string    `json:"id"`
	ClientID  string    `json:"client_id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

type MessageStore struct {
	mu       sync.RWMutex
	messages []Message
}

func NewMessageStore() *MessageStore {
	return &MessageStore{
		messages: make([]Message, 0),
	}
}

// adding messages thread safe
func (ms *MessageStore) Add(msg Message) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.messages = append(ms.messages, msg)
}

// get messages since a given time
func (ms *MessageStore) GetSince(since time.Time) []Message {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	result := make([]Message, 0)
	for _, msg := range ms.messages {
		if msg.CreatedAt.After(since) {
			result = append(result, msg)
		}
	}
	return result
}

type PollingHandler struct {
	registry *registry.ConnectionRegistry
	tracker  *metrics.ProtocolTracker
	store    *MessageStore
}

func NewPollingHandler(reg *registry.ConnectionRegistry, tracker *metrics.ProtocolTracker, store *MessageStore) *PollingHandler {
	return &PollingHandler{
		registry: reg,
		tracker:  tracker,
		store:    store,
	}
}

type PollSendMessageRequest struct {
	ClientID string `json:"client_id"`
	Text     string `json:"text"`
}

type PollSendMessagesResponse struct {
	Status  string  `json:"status"`
	Message Message `json:"message"`
}

func (h *PollingHandler) HandleSend(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PollSendMessageRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid Json payload", http.StatusBadRequest)
		return
	}
	if req.ClientID == "" {
		req.ClientID = "polling-client-default"
	}

	msg := Message{
		ID:        fmt.Sprintf("msg-%d", time.Now().UnixNano()),
		ClientID:  req.ClientID,
		Text:      req.Text,
		CreatedAt: start,
	}

	h.store.Add(msg)
	h.registry.Add(&registry.Client{
		ID:          req.ClientID,
		Protocol:    "polling",
		ConnectedAt: start,
	})

	h.registry.RecordMessage(req.ClientID)
	duration := time.Since(start)
	h.tracker.Record(duration)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(PollSendMessagesResponse{
		Status:  "success",
		Message: msg,
	})

}

type PollGetMessagesResponse struct {
	Status   string    `json:"status"`
	Count    int       `json:"count"`
	Messages []Message `json:"messages"`
}

func (h *PollingHandler) HandleGetMessage(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var sinceTime time.Time
	sinceStr := r.URL.Query().Get("since")
	if sinceStr != "" {
		if ms, err := strconv.ParseInt(sinceStr, 10, 64); err == nil {
			sinceTime = time.UnixMilli(ms)
		} else if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			sinceTime = t
		}
	}

	// 2. Fetch messages from MessageStore created after 'sinceTime'

	messages := h.store.GetSince(sinceTime)

	// 3. Track polling client activity if client_id is passed

	clientID := r.URL.Query().Get("client_id")
	if clientID != "" {
		h.registry.Add(&registry.Client{
			ID:          clientID,
			Protocol:    "polling",
			ConnectedAt: start,
		})
		h.registry.RecordMessage(clientID)
	}

	duration := time.Since(start)
	h.tracker.Record(duration)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(PollGetMessagesResponse{
		Status:   "success",
		Count:    len(messages),
		Messages: messages,
	})

}

func (h *PollingHandler) HandleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	activeConnections := h.registry.Count()
	clients := h.registry.List()
	var totalMessages int64
	for _, client := range clients {
		if client.Protocol == "polling" {
			totalMessages += client.MessageCount
		}
	}
	metricsSnapshot := h.tracker.Snapshot(activeConnections, totalMessages)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(metricsSnapshot)

}


