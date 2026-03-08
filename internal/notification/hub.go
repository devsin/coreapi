package notification

import (
	"encoding/json"
	"sync"
)

// Event is a real-time notification pushed via SSE.
type Event struct {
	Type         string `json:"type"` // "new_notification" | "count_update"
	UnreadCount  int64  `json:"unread_count"`
	Notification *DTO   `json:"notification,omitempty"`
}

// Hub manages per-user SSE connections.
// It is safe for concurrent use.
type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[chan []byte]struct{} // userID → set of channels
}

// NewHub creates a new SSE hub.
func NewHub() *Hub {
	return &Hub{
		clients: make(map[string]map[chan []byte]struct{}),
	}
}

// Subscribe registers a client channel for a user.
// Returns the channel to read from.
func (h *Hub) Subscribe(userID string) chan []byte {
	ch := make(chan []byte, 16)
	h.mu.Lock()
	if h.clients[userID] == nil {
		h.clients[userID] = make(map[chan []byte]struct{})
	}
	h.clients[userID][ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

// Unsubscribe removes a client channel for a user.
func (h *Hub) Unsubscribe(userID string, ch chan []byte) {
	h.mu.Lock()
	if subs, ok := h.clients[userID]; ok {
		delete(subs, ch)
		if len(subs) == 0 {
			delete(h.clients, userID)
		}
	}
	h.mu.Unlock()
	close(ch)
}

// Publish sends an event to all connected clients of a user.
func (h *Hub) Publish(userID string, evt Event) {
	data, err := json.Marshal(evt)
	if err != nil {
		return
	}

	h.mu.RLock()
	subs := h.clients[userID]
	h.mu.RUnlock()

	for ch := range subs {
		select {
		case ch <- data:
		default:
			// Client too slow — drop the message.
		}
	}
}
