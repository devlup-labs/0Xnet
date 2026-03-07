package relay

import (
	"sync"

	"github.com/gorilla/websocket"
)

type Device struct {
	ID   string
	Conn *websocket.Conn
}

type Hub struct {
	Devices map[string]*Device
	mu      sync.Mutex
}

func NewHub() *Hub {
	return &Hub{
		Devices: make(map[string]*Device),
	}
}

func (h *Hub) Register(id string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.Devices[id] = &Device{
		ID:   id,
		Conn: conn,
	}
}

func (h *Hub) Remove(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.Devices, id)
}

func (h *Hub) Broadcast(sender string, msg []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for id, d := range h.Devices {
		if id != sender {
			d.Conn.WriteMessage(websocket.TextMessage, msg)
		}
	}
}
