package handlers

import (
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
