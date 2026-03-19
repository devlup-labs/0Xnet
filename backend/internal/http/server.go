package httpapi

import (
	"database/sql"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/bhawani-prajapat2006/0Xnet/backend/internal/discovery"
	"github.com/bhawani-prajapat2006/0Xnet/backend/internal/websocket"
)

func hashString(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

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
	// Unified Session Router
	http.HandleFunc("/session/", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/session/create":
			if r.Method == http.MethodPost {
				s.createSession(w, r)
			} else {
				http.Error(w, "Use POST", 405)
			}
		case "/session/list":
			if r.Method == http.MethodGet {
				s.listSessions(w, r)
			} else {
				http.Error(w, "Use GET", 405)
			}
		case "/session/delete":
			if r.Method == http.MethodPost {
				s.deleteSession(w, r)
			} else {
				http.Error(w, "Use POST", 405)
			}
		case "/session/join":
			if r.Method == http.MethodPost {
				s.joinSession(w, r)
			} else {
				http.Error(w, "Use POST", 405)
			}
		case "/session/leave":
			if r.Method == http.MethodPost {
				s.leaveSession(w, r)
			} else {
				http.Error(w, "Use POST", 405)
			}
		case "/session/members":
			if r.Method == http.MethodGet {
				s.getSessionMembers(w, r)
			} else {
				http.Error(w, "Use GET", 405)
			}
		default:
			http.NotFound(w, r)
		}
	})

	// Devices Router
	http.HandleFunc("/devices", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		devices := s.sessionDiscovery.GetDiscoveredDevices()
		// Hash device IDs for privacy/security
		hashedDevices := make([]*discovery.DiscoveredDevice, 0, len(devices))
		for _, d := range devices {
			hashedID := hashString(d.DeviceID)
			hashedDevices = append(hashedDevices, &discovery.DiscoveredDevice{DeviceID: hashedID, Address: d.Address, Port: d.Port})
		}
		meHashed := &discovery.DiscoveredDevice{DeviceID: hashString(s.deviceID) + " (Me)"}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(append([]*discovery.DiscoveredDevice{meHashed}, hashedDevices...))
	})

	// Register device via HTTP (useful for browser clients)
	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

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

		// register on discovery
		s.sessionDiscovery.RegisterDevice(body.DeviceID, r.RemoteAddr, 0)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Returns this device's current ID (used by other devices to filter stale sessions)
	http.HandleFunc("/whoami", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"deviceId": s.deviceID})
	})

	http.HandleFunc("/ws", websocket.ServeWS)

	log.Printf("🌍 0Xnet API active on port %d", s.port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", s.port), nil))

}
