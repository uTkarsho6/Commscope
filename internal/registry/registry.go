package registry

import (
	"sync"
	"time"
)

// Client: This holds the state of a single connection. We need to track the protocol type, how many messages they sent (MessageCount), when they connected, and when they were last seen.

type Client struct {
	ID           string
	Protocol     string
	ConnectedAt  time.Time
	MessageCount int64
	LastSeen     time.Time
}

// ConnectionRegistry: This is our in-memory data store. Since multiple client requests and goroutines will read and write to this registry concurrently, we guard the connections map with a Read-Write Mutex (sync.RWMutex) to avoid race conditions.
type ConnectionRegistry struct {
	mu          sync.RWMutex
	connections map[string]*Client
}

// for proper initialization of map; otherwise map will be nil and cause panic when we try to add a client
func NewConnectionRegistry() *ConnectionRegistry {
	return &ConnectionRegistry{
		connections: make(map[string]*Client),
	}
}

// add a client to the registry
// Locking ensures that even if multiple goroutines call Add() at the same time,
// only one can modify the map at any given moment, preventing data corruption.
func (cr *ConnectionRegistry) Add(client *Client) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.connections[client.ID] = client

}

// if key does not exist this does not do anything
func (cr *ConnectionRegistry) Remove(id string) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	delete(cr.connections, id)
}

// count the number of connections
func (cr *ConnectionRegistry) Count() int {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	return len(cr.connections)

}

// List all the connections, return Sclice of clients;
func (cr *ConnectionRegistry) List() []Client {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	clients := make([]Client, 0, len(cr.connections))
	for _, client := range cr.connections {
		clients = append(clients, *client)
	}
	return clients
}

// RecordMessage: Increments the message count for a given client ID. 
// cannot increment the same client's counter at the exact same time.
func (cr *ConnectionRegistry) RecordMessage(id string) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	if client, exists := cr.connections[id]; exists {
		client.MessageCount++
		client.LastSeen = time.Now()

	}
}
