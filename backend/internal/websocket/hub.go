package websocket

import (
	"encoding/json"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload,omitempty"`
}

type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	userID uuid.UUID
	send   chan []byte
}

type Hub struct {
	mu         sync.RWMutex
	clients    map[uuid.UUID]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan *Message
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[uuid.UUID]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *Message, 256),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.userID] = client
			h.mu.Unlock()
			log.Info().Str("user_id", client.userID.String()).Msg("ws client connected")

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.userID]; ok {
				delete(h.clients, client.userID)
				close(client.send)
			}
			h.mu.Unlock()
			log.Info().Str("user_id", client.userID.String()).Msg("ws client disconnected")

		case msg := <-h.broadcast:
			h.mu.RLock()
			data, err := json.Marshal(msg)
			if err != nil {
				h.mu.RUnlock()
				continue
			}
			for _, client := range h.clients {
				select {
				case client.send <- data:
				default:
					close(client.send)
					delete(h.clients, client.userID)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) SendToUser(userID uuid.UUID, msg *Message) {
	h.mu.RLock()
	client, ok := h.clients[userID]
	h.mu.RUnlock()
	if !ok {
		return
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	select {
	case client.send <- data:
	default:
	}
}

func (h *Hub) BroadcastToAll(msg *Message) {
	h.broadcast <- msg
}

func (h *Hub) BroadcastTerritoryChange(msg *Message) {
	h.broadcast <- msg
}

func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
