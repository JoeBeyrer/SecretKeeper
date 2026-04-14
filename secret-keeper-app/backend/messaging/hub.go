package messaging

import (
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	UserID string
	Conn   *websocket.Conn
	Send   chan []byte
}

type Hub struct {
	clients map[string][]*Client // userID -> slice of clients (one per open tab)
	mu      sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[string][]*Client),
	}
}

func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[client.UserID] = append(h.clients[client.UserID], client)
}

// Unregister removes a specific client connection for a user.
// Called with the exact *Client pointer so that only that tab's connection is removed.
func (h *Hub) Unregister(userID string, client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	existing := h.clients[userID]
	updated := existing[:0]
	for _, c := range existing {
		if c != client {
			updated = append(updated, c)
		}
	}
	if len(updated) == 0 {
		delete(h.clients, userID)
	} else {
		h.clients[userID] = updated
	}
}

func (h *Hub) SendToUser(userID string, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, client := range h.clients[userID] {
		client.Send <- message
	}
}
