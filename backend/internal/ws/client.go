package ws

import (
	"log/slog"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the client.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the client.
	pongWait = 40 * time.Second

	// Send pings to client with this period. Must be less than pongWait.
	pingPeriod = 30 * time.Second

	// Maximum message size allowed from client.
	maxMessageSize = 512

	// Send buffer size per client.
	sendBufferSize = 256
)

// Client is a middleman between the WebSocket connection and the hub.
type Client struct {
	hub   *Hub
	conn  *websocket.Conn
	send  chan []byte
	rooms []string
}

// readPump pumps messages from the WebSocket connection to the hub.
// It blocks on ReadMessage to keep the TCP socket alive and detect disconnects.
// The application only reads pong/close frames — clients don't send data messages.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				slog.Warn("ws: unexpected close", "error", err)
			}
			break
		}
	}
}

// writePump pumps messages from the send channel to the WebSocket connection.
// A ticker sends periodic pings to detect dead connections.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel — send close frame.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Coalesce queued messages into the same write for efficiency.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte("\n"))
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
