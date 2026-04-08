package httpapi

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bhawani-prajapat2006/0Xnet/backend/internal/discovery"
	"github.com/bhawani-prajapat2006/0Xnet/backend/internal/streaming"
	"github.com/bhawani-prajapat2006/0Xnet/backend/internal/websocket"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func hashString(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

type Server struct {
	db               *sql.DB
	deviceID         string
	sessionDiscovery *discovery.SessionDiscovery
	port             int
	streamMgr        *streaming.StreamManager
}

func NewServer(db *sql.DB, deviceID string, sessionDiscovery *discovery.SessionDiscovery, port int, streamMgr *streaming.StreamManager) *Server {
	return &Server{
		db:               db,
		deviceID:         deviceID,
		sessionDiscovery: sessionDiscovery,
		port:             port,
		streamMgr:        streamMgr,
	}
}

func (s *Server) Start() {
	mux := http.NewServeMux()

	// Unified Session Router
	mux.HandleFunc("/session/", func(w http.ResponseWriter, r *http.Request) {
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
	mux.HandleFunc("/devices", func(w http.ResponseWriter, r *http.Request) {
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
	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
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
	mux.HandleFunc("/whoami", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"deviceId": s.deviceID})
	})

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		websocket.ServeWS(w, r)
	})

	// ── Streaming Routes ────────────────────────────────────

	// File-upload based stream start (host picks file from browser file picker)
	mux.HandleFunc("/stream/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Use POST", 405)
			return
		}

		reader, err := r.MultipartReader()
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"failed to read multipart request: %s"}`, err.Error()), http.StatusBadRequest)
			return
		}

		var sessionID string
		var savedPath string

		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"failed to read part: %s"}`, err.Error()), http.StatusInternalServerError)
				return
			}

			if part.FormName() == "sessionId" {
				buf, _ := io.ReadAll(part)
				sessionID = string(buf)
			} else if part.FormName() == "file" {
				if sessionID == "" {
					http.Error(w, `{"error":"sessionId must be sent before file field"}`, http.StatusBadRequest)
					return
				}
				
				// Save uploaded file to temp directory
				uploadDir := filepath.Join(os.TempDir(), "0xnet-uploads", sessionID)
				os.MkdirAll(uploadDir, 0755)
				savedPath = filepath.Join(uploadDir, part.FileName())

				dst, err := os.Create(savedPath)
				if err != nil {
					http.Error(w, fmt.Sprintf(`{"error":"failed to save file: %s"}`, err.Error()), http.StatusInternalServerError)
					return
				}

				written, err := io.Copy(dst, part)
				dst.Close()
				if err != nil {
					http.Error(w, fmt.Sprintf(`{"error":"failed to write file: %s"}`, err.Error()), http.StatusInternalServerError)
					return
				}
				log.Printf("📁 [Upload] Saved %s (%d MB) for session %s", part.FileName(), written/(1024*1024), sessionID)
			}
		}

		if sessionID == "" || savedPath == "" {
			http.Error(w, `{"error":"missing sessionId or file field"}`, http.StatusBadRequest)
			return
		}

		// Start ffmpeg on the saved file
		playlistURL, err := s.streamMgr.Start(sessionID, savedPath)
		if err != nil {
			os.Remove(savedPath)
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}

		// Notify all peers in the session that streaming has started
		websocket.GlobalManager.GetHub(sessionID).Broadcast(map[string]string{
			"type":        "stream-started",
			"playlistUrl": playlistURL,
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"playlistUrl": playlistURL})
	})

	mux.HandleFunc("/stream/start", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Use POST", 405)
			return
		}
		var body struct {
			SessionID string `json:"sessionId"`
			FilePath  string `json:"filePath"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.SessionID == "" || body.FilePath == "" {
			http.Error(w, `{"error":"sessionId and filePath required"}`, http.StatusBadRequest)
			return
		}

		playlistURL, err := s.streamMgr.Start(body.SessionID, body.FilePath)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}

		// Notify all peers in the session that streaming has started
		websocket.GlobalManager.GetHub(body.SessionID).Broadcast(map[string]string{
			"type":        "stream-started",
			"playlistUrl": playlistURL,
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"playlistUrl": playlistURL})
	})

	mux.HandleFunc("/stream/stop", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Use POST", 405)
			return
		}
		var body struct {
			SessionID string `json:"sessionId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.SessionID == "" {
			http.Error(w, `{"error":"sessionId required"}`, http.StatusBadRequest)
			return
		}

		s.streamMgr.Stop(body.SessionID)

		// Notify all peers that streaming stopped
		websocket.GlobalManager.GetHub(body.SessionID).Broadcast(map[string]string{
			"type": "stream-stopped",
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "stopped"})
	})

	// Serve HLS segments: /stream/<sessionID>/index.m3u8, /stream/<sessionID>/seg_000.ts, etc.
	mux.HandleFunc("/stream/", func(w http.ResponseWriter, r *http.Request) {
		// Skip the start/stop routes that are handled above
		if r.URL.Path == "/stream/start" || r.URL.Path == "/stream/stop" || r.URL.Path == "/stream/upload" {
			return
		}

		// Extract sessionID from path: /stream/<sessionID>/filename
		parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/stream/"), "/", 2)
		if len(parts) < 2 {
			http.NotFound(w, r)
			return
		}
		sessionID := parts[0]

		outputDir := s.streamMgr.GetOutputDir(sessionID)
		if outputDir == "" {
			http.NotFound(w, r)
			return
		}

		// Set proper content types and disable caching for the playlist
		filename := parts[1]
		if strings.HasSuffix(filename, ".m3u8") {
			w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		} else if strings.HasSuffix(filename, ".ts") {
			w.Header().Set("Content-Type", "video/MP2T")
		}

		// Serve the file from the HLS output directory
		prefix := fmt.Sprintf("/stream/%s/", sessionID)
		http.StripPrefix(prefix, http.FileServer(http.Dir(outputDir))).ServeHTTP(w, r)
	})

	log.Printf("🌍 0Xnet API active on port %d", s.port)
	handler := corsMiddleware(mux)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", s.port), handler))
}
