package ws

import (
	"log/slog"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true }, // CORS handled by middleware
}

// Hub maintains the set of active clients and broadcasts events to rooms.
// Room "global" receives ALL events. Room "{group_id}" receives events for that group only.
type Hub struct {
	mu         sync.RWMutex
	rooms      map[string]map[*Client]bool
	register   chan *Client
	unregister chan *Client
}

// NewHub creates a new Hub.
func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub's event loop. Must be called as a goroutine.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			for _, room := range client.rooms {
				if h.rooms[room] == nil {
					h.rooms[room] = make(map[*Client]bool)
				}
				h.rooms[room][client] = true
			}
			h.mu.Unlock()
			slog.Info("ws: client connected", "rooms", client.rooms)

		case client := <-h.unregister:
			h.mu.Lock()
			for _, room := range client.rooms {
				if clients, ok := h.rooms[room]; ok {
					if _, exists := clients[client]; exists {
						delete(clients, client)
						if len(clients) == 0 {
							delete(h.rooms, room)
						}
					}
				}
			}
			close(client.send)
			h.mu.Unlock()
			slog.Info("ws: client disconnected", "rooms", client.rooms)
		}
	}
}

// Broadcast sends an event to all clients in the specified room.
// Uses RLock — multiple broadcasts can run concurrently without blocking each other.
// Slow clients that can't keep up get their channel closed (evicted).
func (h *Hub) Broadcast(room string, event Event) {
	msg := event.Marshal()
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients := h.rooms[room]
	for client := range clients {
		select {
		case client.send <- msg:
		default:
			// Slow client — evict
			close(client.send)
			delete(clients, client)
			slog.Warn("ws: evicted slow client", "room", room)
		}
	}
}

// BroadcastMulti sends an event to multiple rooms (typically "global" + group_id).
func (h *Hub) BroadcastMulti(rooms []string, event Event) {
	for _, room := range rooms {
		h.Broadcast(room, event)
	}
}

// HandleUpgrade upgrades an HTTP request to a WebSocket connection and registers the client.
// Query parameter ?room= specifies which room(s) to join (comma-separated or default "global").
func (h *Hub) HandleUpgrade(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("ws: upgrade failed", "error", err)
		return
	}

	roomParam := r.URL.Query().Get("room")
	if roomParam == "" {
		roomParam = "global"
	}

	// Support joining multiple rooms: ?room=global,{group_id}
	rooms := splitRooms(roomParam)

	client := &Client{
		hub:   h,
		conn:  conn,
		send:  make(chan []byte, sendBufferSize),
		rooms: rooms,
	}

	h.register <- client

	// Start pumps in goroutines — they own the connection lifecycle
	go client.writePump()
	go client.readPump()
}

// ConnectionCount returns the total number of active connections (for /healthz).
func (h *Hub) ConnectionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	count := 0
	seen := make(map[*Client]bool)
	for _, clients := range h.rooms {
		for c := range clients {
			if !seen[c] {
				seen[c] = true
				count++
			}
		}
	}
	return count
}

func splitRooms(s string) []string {
	var rooms []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			room := s[start:i]
			if room != "" {
				rooms = append(rooms, room)
			}
			start = i + 1
		}
	}
	if len(rooms) == 0 {
		rooms = []string{"global"}
	}
	return rooms
}
