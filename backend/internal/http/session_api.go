package httpapi

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"sort"

	"github.com/bhawani-prajapat2006/0Xnet/backend/internal/models"
	"github.com/bhawani-prajapat2006/0Xnet/backend/internal/service"
)

func (s *Server) createSession(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if body.Name == "" {
		http.Error(w, "Session name is required", http.StatusBadRequest)
		return
	}

	session, err := service.CreateSession(s.db, body.Name, s.deviceID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Enrich with host info and members before returning
	session.HostIP = s.getLocalIP()
	session.HostPort = s.port
	session.Members, _ = service.GetSessionMembers(s.db, session.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

func (s *Server) listSessions(w http.ResponseWriter, r *http.Request) {
	// Get local sessions from database
	localSessions, _ := service.ListSessions(s.db, s.deviceID)

	// Enrich local sessions with host IP, port, and members
	for i := range localSessions {
		localSessions[i].HostIP = s.getLocalIP()
		localSessions[i].HostPort = s.port
		localSessions[i].Members, _ = service.GetSessionMembers(s.db, localSessions[i].ID)
	}

	log.Printf("🔎 listSessions called (source=%s) | local=%d", r.URL.Query().Get("source"), len(localSessions))

	// Check if this is a remote discovery request (from another device scanning)
	// or a local request that wants all LAN sessions
	source := r.URL.Query().Get("source")
	if source == "local" {
		// Only return this device's own sessions (for remote fetching)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(localSessions)
		return
	}

	// Get remote sessions from all discovered devices on the LAN
	// These are inherently "active" — we just fetched them from online devices
	remoteSessions := s.sessionDiscovery.GetRemoteSessions()

	// Combine local + remote sessions
	allSessions := append(localSessions, remoteSessions...)

	// Sort by newest first
	sort.Slice(allSessions, func(i, j int) bool {
		return allSessions[i].CreatedAt.After(allSessions[j].CreatedAt)
	})

	log.Printf("🔎 Returning: local=%d remote=%d total=%d", len(localSessions), len(remoteSessions), len(allSessions))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(allSessions)
}

// filterActiveSessions returns only sessions from online hosts
func (s *Server) filterActiveSessions(sessions []models.Session) []models.Session {
	// Get list of all online device IDs
	onlineDevices := make(map[string]bool)
	onlineDevices[s.deviceID] = true // This device is always online

	for _, device := range s.sessionDiscovery.GetDiscoveredDevices() {
		onlineDevices[device.DeviceID] = true
	}

	// Filter sessions
	activeSessions := make([]models.Session, 0)
	for _, session := range sessions {
		if onlineDevices[session.HostID] {
			activeSessions = append(activeSessions, session)
		}
	}

	return activeSessions
}

func (s *Server) deleteSession(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SessionID string `json:"sessionId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Only allow deleting sessions hosted by this device
	err := service.DeleteSession(s.db, body.SessionID, s.deviceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Session closed"})
}

func (s *Server) listDevices(w http.ResponseWriter, r *http.Request) {
	devices := s.sessionDiscovery.GetDiscoveredDevices()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(devices)
}

// getLocalIP returns the server's local IP address
func (s *Server) getLocalIP() string {
	// Extract from the server's known state
	// The server knows the port, and we can derive the IP
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "localhost"
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}
