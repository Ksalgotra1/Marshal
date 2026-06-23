package sse

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Ksalgotra1/Marshal/internal/realtime"
)

type Handler struct {
	Streams StreamRegistry
}

type StreamRegistry interface {
	RegisterStreamClient(realtime.StreamClient)
	UnregisterStreamClient(realtime.StreamClient)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	if h.Streams == nil {
		http.Error(w, "stream unavailable", http.StatusServiceUnavailable)
		return
	}

	roomParam := r.URL.Query().Get("room")
	if roomParam == "" {
		roomParam = "global"
	}

	client := &Client{
		send:  make(chan []byte, 32),
		rooms: splitRooms(roomParam),
	}

	headers := w.Header()
	headers.Set("Content-Type", "text/event-stream")
	headers.Set("Cache-Control", "no-cache")
	headers.Set("Connection", "keep-alive")
	headers.Set("X-Accel-Buffering", "no")

	h.Streams.RegisterStreamClient(client)
	defer h.Streams.UnregisterStreamClient(client)

	keepAlive := time.NewTicker(25 * time.Second)
	defer keepAlive.Stop()

	fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()

	for {
		select {
		case msg, ok := <-client.send:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-keepAlive.C:
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
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
