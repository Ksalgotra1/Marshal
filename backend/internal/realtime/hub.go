package realtime

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type StreamClient interface {
	Rooms() []string
	Send() chan []byte
	Close()
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type Hub struct {
	mu            sync.RWMutex
	rooms         map[string]map[*Client]bool
	streams       map[string]map[StreamClient]bool
	register      chan *Client
	unregister    chan *Client
	sseRegister   chan StreamClient
	sseUnregister chan StreamClient
	dirtyRooms    map[string]bool
}

func NewHub() *Hub {
	return &Hub{
		rooms:         make(map[string]map[*Client]bool),
		streams:       make(map[string]map[StreamClient]bool),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		sseRegister:   make(chan StreamClient),
		sseUnregister: make(chan StreamClient),
		dirtyRooms:    make(map[string]bool),
	}
}

func (h *Hub) Run() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			for _, room := range client.rooms {
				if h.rooms[room] == nil {
					h.rooms[room] = make(map[*Client]bool)
				}
				h.rooms[room][client] = true
				h.dirtyRooms[room] = true
			}
			h.mu.Unlock()
			slog.Info("ws: client connected", "rooms", client.rooms)

		case client := <-h.sseRegister:
			h.mu.Lock()
			for _, room := range client.Rooms() {
				if h.streams[room] == nil {
					h.streams[room] = make(map[StreamClient]bool)
				}
				h.streams[room][client] = true
				h.dirtyRooms[room] = true
			}
			h.mu.Unlock()
			slog.Info("sse: client connected", "rooms", client.Rooms())

		case client := <-h.unregister:
			h.mu.Lock()
			for _, room := range client.rooms {
				if clients, ok := h.rooms[room]; ok {
					if _, exists := clients[client]; exists {
						delete(clients, client)
						h.dirtyRooms[room] = true
						if len(clients) == 0 {
							delete(h.rooms, room)
						}
					}
				}
			}
			close(client.send)
			h.mu.Unlock()
			slog.Info("ws: client disconnected", "rooms", client.rooms)

		case client := <-h.sseUnregister:
			h.mu.Lock()
			for _, room := range client.Rooms() {
				if clients, ok := h.streams[room]; ok {
					if _, exists := clients[client]; exists {
						delete(clients, client)
						h.dirtyRooms[room] = true
						if len(clients) == 0 {
							delete(h.streams, room)
						}
					}
				}
			}
			client.Close()
			h.mu.Unlock()
			slog.Info("sse: client disconnected", "rooms", client.Rooms())

		case <-ticker.C:
			h.mu.Lock()
			dirty := make([]string, 0, len(h.dirtyRooms))
			for room := range h.dirtyRooms {
				dirty = append(dirty, room)
			}
			// Clear dirty flag for next tick
			h.dirtyRooms = make(map[string]bool)
			h.mu.Unlock()

			if len(dirty) > 0 {
				count := h.ConnectionCount()
				event := SystemConnections(count)
				for _, room := range dirty {
					h.Broadcast(room, event)
				}
			}
		}
	}
}

func (h *Hub) Broadcast(room string, event Event) {
	msg := event.Marshal()
	h.mu.Lock()
	defer h.mu.Unlock()

	clients := h.rooms[room]
	for client := range clients {
		select {
		case client.send <- msg:
		default:
			close(client.send)
			delete(clients, client)
			slog.Warn("ws: evicted slow client", "room", room)
		}
	}

	streams := h.streams[room]
	for client := range streams {
		select {
		case client.Send() <- msg:
		default:
			client.Close()
			delete(streams, client)
			slog.Warn("sse: evicted slow client", "room", room)
		}
	}
}

func (h *Hub) BroadcastMulti(rooms []string, event Event) {
	for _, room := range rooms {
		h.Broadcast(room, event)
	}
}

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

	client := &Client{
		hub:   h,
		conn:  conn,
		send:  make(chan []byte, sendBufferSize),
		rooms: splitRooms(roomParam),
	}

	h.register <- client
	go client.writePump()
	go client.readPump()
}

func (h *Hub) RegisterStreamClient(client StreamClient) {
	h.sseRegister <- client
}

func (h *Hub) UnregisterStreamClient(client StreamClient) {
	h.sseUnregister <- client
}

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
	seenStreams := make(map[StreamClient]bool)
	for _, clients := range h.streams {
		for c := range clients {
			if !seenStreams[c] {
				seenStreams[c] = true
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
