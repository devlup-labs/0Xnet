package httpapi

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
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
	// Unified Session Router
	http.HandleFunc("/session/", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/session/create":
			if r.Method == http.MethodPost { s.createSession(w, r) } else { http.Error(w, "Use POST", 405) }
		case "/session/list":
			if r.Method == http.MethodGet { s.listSessions(w, r) } else { http.Error(w, "Use GET", 405) }
		case "/session/delete":
			if r.Method == http.MethodPost { s.deleteSession(w, r) } else { http.Error(w, "Use POST", 405) }
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
		me := &discovery.DiscoveredDevice{DeviceID: s.deviceID + " (Me)"}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(append([]*discovery.DiscoveredDevice{me}, devices...))
	})

	http.HandleFunc("/ws", websocket.ServeWS)

	log.Printf("🌍 0Xnet API active on port %d", s.port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", s.port), nil))
}

// Ensure these functions in your code DO NOT check for r.Method anymore
func (s *Server) createSession(w http.ResponseWriter, r *http.Request) { /* DB logic only */ }
func (s *Server) listSessions(w http.ResponseWriter, r *http.Request)   { /* DB logic only */ }
func (s *Server) deleteSession(w http.ResponseWriter, r *http.Request) { /* DB logic only */ }