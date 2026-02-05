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
	// Existing session handlers
	http.HandleFunc("/session/create", s.createSession)
	http.HandleFunc("/session/list", s.listSessions)
	http.HandleFunc("/session/delete", s.deleteSession)

	// Updated Devices handler
	http.HandleFunc("/devices", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// 1. Get discovered peers
		devices := s.sessionDiscovery.GetDiscoveredDevices()

		// 2. Add "Self" to the list so you can see your own PeerID
		me := &discovery.DiscoveredDevice{
			DeviceID: s.deviceID + " (Me)",
		}
		
		allDevices := append([]*discovery.DiscoveredDevice{me}, devices...)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(allDevices)
	})

	http.HandleFunc("/ws", websocket.ServeWS)

	addr := fmt.Sprintf("0.0.0.0:%d", s.port)
	log.Printf("🌍 API Server listening on %s\n", addr)
	log.Printf("🔗 Check your devices at: http://localhost:%d/devices\n", s.port)
	log.Fatal(http.ListenAndServe(addr, nil))
}