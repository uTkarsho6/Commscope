package handlers

import (
	"commscope/internal/metrics"
	"commscope/internal/registry"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// configure the WebSocket Upgrader

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024, //1kb
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
		//Allow all origins for local playground testing, in production you shouldn't allow any origin
	},
}

// WSClient represents a single connected WebSocket client session.

type WSClient struct {
	ID   string
	Conn *websocket.Conn
	Send chan []byte //outgoing message queue
	Hub  *WSHub
}

// WSHub maintains active connections and manages thread-safe broadcasting.

type WSHub struct {
	clients    map[*WSClient]bool
	broadcast  chan []byte
	register   chan *WSClient
	unregister chan *WSClient
	mu         sync.RWMutex
}

// NewWSHub initializes and returns a new WSHub instance.
func NewWSHub() *WSHub {
	return &WSHub{
		clients:    make(map[*WSClient]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *WSClient),
		unregister: make(chan *WSClient),
		mu:         sync.RWMutex{},
	}

}
func (h *WSHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()

			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.Lock() // // Full Write Lock because delete() mutates the map!

			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					// instead of allowing a process to block here delete it from active client list
					close(client.Send)
					delete(h.clients, client)
				}
			}
			h.mu.Unlock()
		}
	}
}

const (
	writeWait      = 10 * time.Second    // max amt of time server allows for write operations
	pongWait       = 60 * time.Second    // controls how long the server waits for the client's Pong/other read activity before the read deadline expires.
	pingPeriod     = (pongWait * 9) / 10 // server sends a ping every 54 seconds
	maxMessageSize = 512                 // limits the size of an incoming WebSocket message to 512 bytes.
)

//readPump pumps messages from websocket conection to the hub

func (c *WSClient) readPump() {
	defer func() {
		c.Hub.unregister <- c // remove from active list
		c.Conn.Close()        // close the connection
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait)) //60s

	// after receiving pong from client server extends the deadline by 60s
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait)) // newdeadline = currtime + 60s
		return nil                                       // no error
	})

	for {
		_, message, err := c.Conn.ReadMessage() // waits for the next WebSocket message from the client. msg type is ignored
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("websocket read error %v", err)
			}
			break
		}
		c.Hub.broadcast <- message

	}
}

// writePump pumps messages from the hub to tthe websocket connection.

func (c *WSClient) writePump() {

	//ticker produces a ping signal every 54 seconds.
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait)) // 10s

			if !ok { // iff channel is closed
				// The HUB close the channel
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			//Creates a "Web socket message writer" for the next message.
			// to avoid creating overhead of sending multiple websocket frames
			// multiple msg -> combine -> single msg
			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			w.Write(message)

			// Add queued messages to the current message websocket frame
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'}) // write new line
				w.Write(<-c.Send)     // take msg out of channel and write it in current WebSocket message
			}

			if err := w.Close(); err != nil { // Finish this WebSocket message and send/finalize it
				return
			}

		case <-ticker.C: // execute every ~54s
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))                      // set the deadline for 10s (write timeout)
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil { // send the WebSocket Ping frame
				return

			}

		}
	}

}

// WSHandler manages HTTP upgrade endpoints and WebSocket telemetry.

type WSHandler struct {
	Hub      *WSHub
	Registry *registry.ConnectionRegistry
	Tracker  *metrics.ProtocolTracker
}

// NewWSHandler initializes a new WSHandler instance.
func NewWSHandler(hub *WSHub, reg *registry.ConnectionRegistry, tracker *metrics.ProtocolTracker) *WSHandler {
	return &WSHandler{
		Hub:      hub,
		Registry: reg,
		Tracker:  tracker,
	}
}

// HandleConnect upgrades the HTTP connection to a WebSocket connection

func (h *WSHandler) HandleConnect(w http.ResponseWriter, r *http.Request) {

	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		clientID = "ws-client-" + time.Now().Format("150405.000")
	}

	//Converts the normal HTTP connection into a WebSocket connection.
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed %v", err)
		return
	}

	h.Registry.Add(&registry.Client{
		ID:          clientID,
		Protocol:    "ws",
		ConnectedAt: time.Now(),
		LastSeen:    time.Now(),
	})

	client := &WSClient{
		ID:   clientID,
		Conn: conn,
		Send: make(chan []byte, 256),
		Hub:  h.Hub,
	}
	client.Hub.register <- client // add this to active list

	// start goroutines
	go client.writePump() // s->c
	go client.readPump()  // c->s

}

// HandleSend allows broadcasting messages via HTTP POST across connected WebSocket clients.
