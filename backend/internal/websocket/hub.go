package websocket

import (
	"sync"
)

type SessionHub struct {
	SessionID string
	Clients   map[*Client]bool
	mutex     sync.RWMutex
}

func NewSessionHub(id string) *SessionHub {
	return &SessionHub{
		SessionID: id,
		Clients:   make(map[*Client]bool),
	}
}

func (h *SessionHub) Register(c *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.Clients[c] = true
}

func (h *SessionHub) Unregister(c *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	delete(h.Clients, c)
}

func (h *SessionHub) Broadcast(msg interface{}) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	for client := range h.Clients {
		client.Conn.WriteJSON(msg)
	}
}

func (h *SessionHub) BroadcastExcluding(msg interface{}, exclude *Client) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	for client := range h.Clients {
		if client == exclude {
			continue
		}
		client.Conn.WriteJSON(msg)
	}
}

func (h *SessionHub) SendToDevice(deviceID string, msg interface{}) bool {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	for client := range h.Clients {
		if client.DeviceID == deviceID {
			client.Conn.WriteJSON(msg)
			return true
		}
	}

	return false
}

type SessionManager struct {
	Hubs  map[string]*SessionHub
	mutex sync.RWMutex
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		Hubs: make(map[string]*SessionHub),
	}
}

func (m *SessionManager) GetHub(sessionID string) *SessionHub {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if hub, ok := m.Hubs[sessionID]; ok {
		return hub
	}
	hub := NewSessionHub(sessionID)
	m.Hubs[sessionID] = hub
	return hub
}

var GlobalManager = NewSessionManager()
