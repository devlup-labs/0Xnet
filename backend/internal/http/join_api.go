package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/bhawani-prajapat2006/0Xnet/backend/internal/service"
	"github.com/bhawani-prajapat2006/0Xnet/backend/internal/websocket"
)

// joinSession handles POST /session/join
// Allows a device on the LAN to join an existing session
func (s *Server) joinSession(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SessionID  string `json:"sessionId"`
		DeviceID   string `json:"deviceId"`
		DeviceName string `json:"deviceName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if body.SessionID == "" || body.DeviceID == "" {
		http.Error(w, "sessionId and deviceId are required", http.StatusBadRequest)
		return
	}

	if body.DeviceName == "" {
		body.DeviceName = body.DeviceID // fallback to deviceId as name
	}

	member, err := service.JoinSession(s.db, body.SessionID, body.DeviceID, body.DeviceName)
	if err != nil {
		http.Error(w, "Failed to join session: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "joined",
		"member": member,
	})
}

// leaveSession handles POST /session/leave
// Allows a device to leave a session it previously joined
func (s *Server) leaveSession(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SessionID string `json:"sessionId"`
		DeviceID  string `json:"deviceId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if body.SessionID == "" || body.DeviceID == "" {
		http.Error(w, "sessionId and deviceId are required", http.StatusBadRequest)
		return
	}

	// Check if this is the host leaving
	isHost := service.IsHost(s.db, body.SessionID, body.DeviceID)

	if isHost {
		// Notify all guests that the host is leaving — they have 10 seconds
		log.Printf("🔔 Host %s leaving session %s — notifying guests", body.DeviceID, body.SessionID)
		websocket.GlobalManager.GetHub(body.SessionID).Broadcast(map[string]interface{}{
			"type":       "host-left",
			"countdown":  10,
			"message":    "Host has left. Session ending in 10 seconds…",
		})

		// Stop any active media stream
		s.streamMgr.Stop(body.SessionID)

		// Schedule cleanup after 10 seconds so guests have time to see the message
		go func() {
			time.Sleep(10 * time.Second)
			// Broadcast final session-ended event
			websocket.GlobalManager.GetHub(body.SessionID).Broadcast(map[string]interface{}{
				"type":    "session-ended",
				"message": "Session has been closed by the host.",
			})
			log.Printf("🧹 Cleaning up session %s after host departure", body.SessionID)
		}()
	}

	sessionDeleted, err := service.LeaveSession(s.db, body.SessionID, body.DeviceID)
	if err != nil {
		http.Error(w, "Failed to leave session: "+err.Error(), http.StatusNotFound)
		return
	}

	status := "left"
	if sessionDeleted {
		status = "session_deleted"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": status,
	})
}

// getSessionMembers handles GET /session/members?sessionId=X
// Returns all devices that have joined a specific session
func (s *Server) getSessionMembers(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		http.Error(w, "sessionId query parameter is required", http.StatusBadRequest)
		return
	}

	members, err := service.GetSessionMembers(s.db, sessionID)
	if err != nil {
		http.Error(w, "Failed to get members: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(members)
}
