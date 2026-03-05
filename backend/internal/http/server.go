package httpapi

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/bhawani-prajapat2006/0Xnet/backend/internal/discovery"
	"github.com/bhawani-prajapat2006/0Xnet/backend/internal/websocket"
)

type Server struct {
	db               *sql.DB
	deviceID         string
	sessionDiscovery *discovery.SessionDiscovery
	port             int
}

func NewServer(db *sql.DB, deviceID string, sessionDiscovery *discovery.SessionDiscovery, port int) *Server {
	return &Server{
		db:               db,
		deviceID:         deviceID,
		sessionDiscovery: sessionDiscovery,
		port:             port,
	}
}

func (s *Server) Start() {
	http.HandleFunc("/session/create", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.createSession(w, r)
	})

	http.HandleFunc("/session/list", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// If a remote browser/mobile calls this endpoint, auto-register it
		// so it shows up in the devices list for testing.
		remoteAddr := r.RemoteAddr
		if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
			// Build a simple device id from the remote host
			deviceID := "browser-" + host

			// Check if already registered
			exists := false
			for _, d := range s.sessionDiscovery.GetDiscoveredDevices() {
				if d.DeviceID == deviceID {
					exists = true
					break
				}
			}
			if !exists {
				s.sessionDiscovery.RegisterDevice(deviceID)
				log.Printf("Registered HTTP client device: %s", deviceID)
			}
		}

		devices := s.sessionDiscovery.GetDiscoveredDevices()
		me := &discovery.DiscoveredDevice{DeviceID: s.deviceID + " (Me)"}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(append([]*discovery.DiscoveredDevice{me}, devices...))
	})

	http.HandleFunc("/devices", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodGet {
			// List all discovered devices
			devices := s.sessionDiscovery.GetDiscoveredDevices()
			me := &discovery.DiscoveredDevice{DeviceID: s.deviceID + " (Me)"}
			json.NewEncoder(w).Encode(append([]*discovery.DiscoveredDevice{me}, devices...))
			return
		}

		if r.Method == http.MethodPost {
			// Register a new device
			var body struct {
				DeviceID string `json:"device_id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "invalid body", http.StatusBadRequest)
				return
			}
			if body.DeviceID == "" {
				http.Error(w, "device_id required", http.StatusBadRequest)
				return
			}
			s.sessionDiscovery.RegisterDevice(body.DeviceID)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			return
		}

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})

	http.HandleFunc("/ws", websocket.ServeWS)

	addr := fmt.Sprintf("0.0.0.0:%d", s.port)
	log.Printf("Server listening on %s\n", addr)
	http.ListenAndServe(addr, nil)
}
