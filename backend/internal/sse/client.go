package sse

// Client is a one-way live subscriber used by SSE clients.
type Client struct {
	send  chan []byte
	rooms []string
}

func (c *Client) Rooms() []string { return c.rooms }

func (c *Client) Send() chan []byte { return c.send }

func (c *Client) Close() { close(c.send) }
