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

type LongPollBroadcaster struct {
	mu sync.RWMutex
	listeners map[chan Message] struct{}
}

func NewLongPollBroadcaster() *LongPollBroadcaster {
	return &LongPollBroadcaster{
		listeners: make(map[chan Message] struct{}),
	}
}

func (b *LongPollBroadcaster) Subscribe() chan Message {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan Message, 1)

	b.listeners[ch] = struct{}{}

	return ch
}

func (b *LongPollBroadcaster) Unsubscribe(ch chan Message){
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.listeners, ch)

}

func (b *LongPollBroadcaster) Broadcast(msg Message) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch:= range b.listeners {
		select {
		case ch <- msg: default:
			      // Non-blocking send: prevents
				  //  a slow/stuck listener from blocking the broadcast
		}
	}
}

type LongPollingHandler struct {
	registry *registry.ConnectionRegistry 
	tracker *metrics.ProtocolTracker
	store *MessageStore
	broadcaster *LongPollBroadcaster
}

func NewLongPollingHandler(reg *registry.ConnectionRegistry, tracker *metrics.ProtocolTracker, store *MessageStore, broadcaster *LongPollBroadcaster) *LongPollingHandler{
	return &LongPollingHandler{
		registry : reg,
		tracker : tracker,
		store : store,
		broadcaster : broadcaster,
	}
}

//Implement HandleSend on LongPollingHandler to accept a new message, 
// save it to MessageStore, broadcast it to all hanging long-polling listener channels via h.broadcaster.Broadcast(msg),
//  record client activity & telemetry, and return JSON.

type LongPollSendMessageRequest struct {
	ClientID string `json:"client_id"`
	Text string `json:"text"`
}

type LongPollSendMessageResponse struct {
    
		Status string `json:"status"`
		Message Message `json:"message"`
	}


func (h *LongPollingHandler) HandleSend(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LongPollSendMessageRequest 
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w,"Invalid JSON payload", http.StatusBadRequest)
		return
	}

	if req.ClientID == ""{
		req.ClientID = "longpolling-client-default"
	}

	msg := Message{
		ID: fmt.Sprintf("msg-%d", time.Now().UnixNano()),
		ClientID: req.ClientID,
		Text: req.Text,
		CreatedAt: start,
	}

	h.store.Add(msg) // store msg
	h.broadcaster.Broadcast(msg) // Broadcast to ALL GET listeners
	// Track registry and telemetry
	h.registry.Add(&registry.Client{
		ID: req.ClientID,
		Protocol: "longpolling",
		ConnectedAt: start,
	})
	
	h.registry.RecordMessage(req.ClientID)
	duration := time.Since(start)

	h.tracker.Record(duration)
	 // send JSON response

	 w.Header().Set("Content-Type", "application/json")
	 w.WriteHeader(http.StatusOK)

	 json.NewEncoder(w).Encode(LongPollSendMessageResponse{
		Status: "success",
		Message: msg,
	 })

}

// Hanging GET request

type LongPollGetMessagesResponse struct {
	Status string `json:"status"`
	Count int `json:"count"`
	Messages []Message `json:"messages"`
}

func (h *LongPollingHandler) HandleGetMessages (w http.ResponseWriter, r *http.Request){
	start := time.Now()

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	} 

	// parse 'since' query parameter

	var sinceTime time.Time
	sinceStr := r.URL.Query().Get("since")
	if sinceStr != ""{
		if ms, err := strconv.ParseInt(sinceStr,10,64); err == nil {
			sinceTime = time.UnixMilli(ms)
		} else if t,err := time.Parse(time.RFC3339,sinceStr); err == nil {
			sinceTime = t
		}
	}

	   // 2. Parse 'timeout' query parameter (default 30 seconds)
    timeoutSec := 30
	if tStr := r.URL.Query().Get("timeout"); tStr != ""{
		if tSec, err := strconv.Atoi(tStr); err == nil && tSec > 0 {
			timeoutSec = tSec
		}
	}

    timeoutDuration := time.Duration(timeoutSec) * time.Second
    
	// 3. Check if there are new messages immediately
    messages := h.store.GetSince(sinceTime)

	 // 4. If NO new messages exist, HANG/WAIT until broadcast or timeout!
     
	 if len(messages) ==0{
		ch := h.broadcaster.Subscribe()
		defer h.broadcaster.Unsubscribe(ch)

		select {
		case msg := <- ch: messages = append(messages, msg)
		case <- time.After(timeoutDuration): 
		       // Timeout reached: returns empty array []
	    case <- r.Context().Done(): //client disconnected
		  return
		}
	 }

	 // 5. Track registry(client) and telemetry
	 
	 clientID := r.URL.Query().Get("client_id")
	 if clientID != ""{
		h.registry.Add(&registry.Client{
			ID: clientID,
			Protocol: "longpolling",
			ConnectedAt: start,
		})
		h.registry.RecordMessage(clientID)

	 }
	 duration := time.Since(start)
	 h.tracker.Record(duration)

	 // 6. Send JSON response
	 w.Header().Set("Content-Type", "application/json")

	 w.WriteHeader(http.StatusOK)

	 json.NewEncoder(w).Encode(LongPollGetMessagesResponse{
		Status : "success",
		Count: len(messages),
		Messages: messages,
	 })

}

func (h *LongPollingHandler) HandleStats(w http.ResponseWriter, r *http.Request){
	if r.Method != http.MethodGet {
     http.Error(w,"Method not allowed", http.StatusMethodNotAllowed)
	 return
	}
	activeConnections := h.registry.Count()
	clients := h.registry.List()
	var totalMessages int64

	for _, client := range clients {
		if client.Protocol == "longpolling"{
			totalMessages += client.MessageCount
		}
	}
	metricsSnapshot := h.tracker.Snapshot(activeConnections, totalMessages)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(metricsSnapshot)
}

