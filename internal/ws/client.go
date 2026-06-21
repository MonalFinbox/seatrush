package ws

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10 // ping before the pong deadline elapses
	sendBuffer = 16
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Dev-only: accept any origin. In production this would whitelist the
	// frontend's domain.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Client is one WebSocket connection subscribed to one event.
type Client struct {
	hub     *Hub
	conn    *websocket.Conn
	eventID string
	send    chan []byte
}

// Serve upgrades the HTTP request to a WebSocket and runs the read/write pumps
// for a client subscribed to eventID.
func Serve(hub *Hub, eventID string, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return // Upgrade already wrote an error response
	}
	c := &Client{
		hub:     hub,
		conn:    conn,
		eventID: eventID,
		send:    make(chan []byte, sendBuffer),
	}
	hub.register(c)

	go c.writePump()
	c.readPump() // blocks until the connection closes
}

// readPump drains incoming frames. Clients are receive-only here, so we just
// keep the connection healthy (pong handler) and detect disconnects.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister(c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}

// writePump sends queued messages and periodic pings to keep the socket alive.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
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
