package ws

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = 25 * time.Second
	maxMessageSize = 4096
	sendBuffer     = 64
)

// Client WebSocket 连接
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	roomID   string
	clientID string
	userID   uint64
	lastSeq  uint64
	mu       sync.Mutex
}

func (c *Client) setLastSeq(seq uint64) {
	c.mu.Lock()
	if seq > c.lastSeq {
		c.lastSeq = seq
	}
	c.mu.Unlock()
}

func (c *Client) getLastSeq() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lastSeq
}

func (c *Client) sendJSON(env Envelope) {
	b, err := json.Marshal(env)
	if err != nil {
		return
	}
	select {
	case c.send <- b:
	default:
		log.Printf("ws: client %s send buffer full, dropping", c.clientID)
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.removeClient(c)
		_ = c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		c.hub.handleClientMessage(c, data)
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
