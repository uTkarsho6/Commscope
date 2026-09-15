package webrtcserver

import (
	"commscope/internal/metrics"
	"commscope/internal/registry"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/pion/webrtc/v3"
)

// SignalMessage represents an SDP Offer/Answer or ICE Candidate exchange payload.
type SignalMessage struct {
	Type      string `json:"type"` // offer, answer, ice-candidate
	PeerID    string `json:"peer_id"`
	SDP       string `json:"sdp"`       // Session Description Protocol
	Candidate string `json:"candidate"` // ICE Candidate
	Text      string `json:"text"`      // data channel text payload
}

// SessionStore maintains active WebRTC Peer connections and data channels

type SessionStore struct {
	mu          sync.RWMutex
	connections map[string]*webrtc.PeerConnection
	channels    map[string]*webrtc.DataChannel
}

// NewSessionStore initializes a new SessionStore instance.
func NewSessionStore() *SessionStore {
	return &SessionStore{
		connections: make(map[string]*webrtc.PeerConnection),
		channels:    make(map[string]*webrtc.DataChannel),
	}
}

// WebRTCHandler manages WebRTC signaling endpoints and DataChannel telemetry.
type WebRTCHandler struct {
	Sessions *SessionStore
	Registry *registry.ConnectionRegistry
	Tracker  *metrics.ProtocolTracker
}

// NewWebRTCHandler initializes a new WebRTCHandler
func NewWebRTCHandler(reg *registry.ConnectionRegistry, tracker *metrics.ProtocolTracker) *WebRTCHandler {
	return &WebRTCHandler{
		Sessions: NewSessionStore(),
		Registry: reg,
		Tracker:  tracker,
	}
}

// HandleOffer receives an SDP Offer, sets up PeerConnection & DataChannel, and returns an SDP Answer.
func (h *WebRTCHandler) HandleOffer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var msg SignalMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "Invalid json payload", http.StatusBadRequest)
		return
	}
	peerID := msg.PeerID
	if peerID == "" {
		peerID = fmt.Sprintf("webrtc-peer-%d", time.Now().UnixNano())
	}

	// 1. Configure WebRTC PeerConnection with public STUN server
	config := webrtc.Configuration{
		// ICEServers defines a slice describing servers available to be used by
		// ICE, such as STUN and TURN servers.
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create PeerConnection: %v", err), http.StatusInternalServerError)
		return
	}

	// 2. Register connection in SessionStore & ConnectionRegistry
	h.Sessions.mu.Lock()
	h.Sessions.connections[peerID] = peerConnection
	h.Sessions.mu.Unlock()

	h.Registry.Add(&registry.Client{
		ID:          peerID,
		Protocol:    "webrtc",
		ConnectedAt: time.Now(),
		LastSeen:    time.Now(),
	})

	// 3. Handle DataChannel creation from remote peer
	// OnDataChannel sets an event handler which is invoked when a data channel message arrives from a remote peer.
	peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		h.Sessions.mu.Lock()
		h.Sessions.channels[peerID] = d
		h.Sessions.mu.Unlock()

		d.OnOpen(func() {
			log.Printf("DataChannel '%s'-'%d' open for peer %s\n", d.Label(), d.ID(), peerID)
		})

		//OnMessage sets an event handler which is invoked on a binary message arrival over the sctp transport from a remote peer.
		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			start := time.Now()
			h.Registry.RecordMessage(peerID)
			duration := time.Since(start)
			h.Tracker.Record(duration)

			// Echo message back over P2P DataChannel, because:
			_ = d.SendText(fmt.Sprintf("P2P Echo: %s", string(msg.Data)))
		})
	})

	// 4. Set Remote Description (SDP Offer)
	offer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  msg.SDP,
	}

	if err := peerConnection.SetRemoteDescription(offer); err != nil {
		http.Error(w, fmt.Sprintf("Failed to set remote description: %v", err), http.StatusInternalServerError)
		return
	}

	// 5. Create SDP Answer
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create answer: %v", err), http.StatusInternalServerError)
		return
	}

	// 6. Set Local Description
	if err := peerConnection.SetLocalDescription(answer); err != nil {
		http.Error(w, fmt.Sprintf("Failed to set local description: %v", err), http.StatusInternalServerError)
		return
	}

	// 7. Send Answer back to client as JSON
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(SignalMessage{
		Type:   "answer",
		PeerID: peerID,
		SDP:    answer.SDP,
	})
}

// HandleStats returns JSON snapshot of WebRTC connections and metrics telemetry.

func (h *WebRTCHandler) HandleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.Sessions.mu.RLock()
	activeConns := len(h.Sessions.connections)
	h.Sessions.mu.RUnlock()

	clients := h.Registry.List()
	var totalMessages int64
	for _, client := range clients {
		if client.Protocol == "webrtc" {
			totalMessages += client.MessageCount
		}
	}

	snapshot := h.Tracker.Snapshot(activeConns, totalMessages)


	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(snapshot)

}
