// WebSocket hub: fan-out of public market data (chapter 10). Slow clients
// never block the dispatcher — each client has a bounded queue and is dropped
// when it falls too far behind (backpressure by disconnection).
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/coder/websocket"

	"github.com/mohammadzeyghami/go-mini-exchange/internal/domain"
)

type wsMsg struct {
	Type string          `json:"type"` // trade | depth | order | ticker
	Data json.RawMessage `json:"data"`
}

type client struct {
	send      chan []byte
	closeOnce sync.Once
}

func (c *client) drop() { c.closeOnce.Do(func() { close(c.send) }) }

type Hub struct {
	mu      sync.Mutex
	clients map[*client]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[*client]struct{})}
}

func (h *Hub) broadcast(typ string, data any) {
	raw, err := json.Marshal(data)
	if err != nil {
		return
	}
	msg, _ := json.Marshal(wsMsg{Type: typ, Data: raw})
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		select {
		case c.send <- msg:
		default: // client too slow — drop it, never block the producer
			c.drop()
			delete(h.clients, c)
		}
	}
}

func (h *Hub) BroadcastTrade(t domain.Trade, lastPrice int64) {
	h.broadcast("trade", t)
	h.broadcast("ticker", map[string]int64{"last": lastPrice})
}

func (h *Hub) BroadcastDepth(d domain.Depth) { h.broadcast("depth", d) }

func (h *Hub) BroadcastOrder(o domain.Order) { h.broadcast("order", o) }

// Serve upgrades the connection and streams until the client goes away.
func (h *Hub) Serve(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"}, // internal demo tool — no cookies, no auth
	})
	if err != nil {
		return
	}
	c := &client{send: make(chan []byte, 256)}
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.clients, c)
		h.mu.Unlock()
		c.drop()
		conn.Close(websocket.StatusNormalClosure, "")
	}()

	ctx := r.Context()
	// Reader goroutine: we ignore client messages but must drain to detect close.
	go func() {
		for {
			if _, _, err := conn.Read(context.Background()); err != nil {
				return
			}
		}
	}()
	for msg := range c.send {
		if err := conn.Write(ctx, websocket.MessageText, msg); err != nil {
			return
		}
	}
}
