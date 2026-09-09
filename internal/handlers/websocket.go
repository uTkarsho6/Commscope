package handlers

import (
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// configure the WebSocket Upgrader

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,  //1kb
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
	Send chan []byte      //outgoing message queue
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


