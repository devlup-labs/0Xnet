package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/bhawani-prajapat2006/0Xnet/backend/internal/models"
	"github.com/bhawani-prajapat2006/0Xnet/backend/internal/service"
)

func (s *Server) createSession(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	session, err := service.CreateSession(s.db, body.Name, s.deviceID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(session)
}

func (s *Server) listSessions(w http.ResponseWriter, r *http.Request) {
	// Get local sessions from database
	localSessions, _ := service.ListSessions(s.db)
	
	// Get sessions from all devices on the LAN
	allSessions := s.sessionDiscovery.GetAllSessions(localSessions)
	
	// Filter: Only show sessions where host device is online (discovered) or local
	activeSessions := s.filterActiveSessions(allSessions)
	
	json.NewEncoder(w).Encode(activeSessions)
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
	json.NewDecoder(r.Body).Decode(&body)

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
