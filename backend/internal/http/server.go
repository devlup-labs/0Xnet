package httpapi

import (
	"database/sql"
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
		s.listSessions(w, r)
	})
	
	http.HandleFunc("/devices", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.listDevices(w, r)
	})
	
	http.HandleFunc("/ws", websocket.ServeWS)
	
	addr := fmt.Sprintf("0.0.0.0:%d", s.port)
	log.Printf("Server listening on %s\n", addr)
	http.ListenAndServe(addr, nil)
}
