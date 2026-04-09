package websocket

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type Client struct {
	DeviceID string // Can be username or peer ID
	Conn     *websocket.Conn
	Session  string
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func ServeWS(w http.ResponseWriter, r *http.Request, onJoin func(*Client)) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WS Upgrade Error: %v", err)
		return
	}
	defer conn.Close()

	// 1. Authenticate / Identify on Join
	var initialMsg map[string]string
	if err := conn.ReadJSON(&initialMsg); err != nil {
		log.Printf("WS Initial Read Error: %v", err)
		return
	}

	if initialMsg["type"] != "join-session" || initialMsg["sessionId"] == "" {
		log.Println("WS Rejected: Invalid join-session message")
		return
	}

	sessionID := initialMsg["sessionId"]
	username := initialMsg["username"]
	if username == "" {
		username = "Anonymous"
	}

	client := &Client{
		DeviceID: username,
		Conn:     conn,
		Session:  sessionID,
	}

	hub := GlobalManager.GetHub(sessionID)
	hub.Register(client)
	defer hub.Unregister(client)

	log.Printf("WS Client Connected: %s to Session %s", username, sessionID)

	// Notify others of new join (optional, good for status)
	hub.Broadcast(map[string]interface{}{
		"type":    "system",
		"message": username + " joined the session",
	})

	if onJoin != nil {
		onJoin(client)
	}

	// 2. Main Message Loop
	for {
		var incoming map[string]interface{}
		err := conn.ReadJSON(&incoming)
		if err != nil {
			log.Printf("WS Read Error: %v", err)
			break
		}

		msgType, _ := incoming["type"].(string)

		switch msgType {
		case "chat":
			hub.Broadcast(map[string]interface{}{
				"type":      "chat",
				"sender":    username,
				"message":   incoming["message"],
				"timestamp": incoming["timestamp"],
			})

		case "sync-playback":
			// Broadcast the sync object to everyone except the sender natively (the frontend handles ignore if it wants, but we broadcast it)
			incoming["sender"] = username
			hub.Broadcast(incoming)

		case "offer", "answer", "ice-candidate", "renegotiate":
			// WebRTC Signaling: Relay only to the intended peer.
			targetPeerID, _ := incoming["targetPeerId"].(string)
			if targetPeerID == "" {
				log.Printf("WS Signaling Missing targetPeerId | type=%s sender=%s", msgType, client.DeviceID)
				continue
			}

			incoming["sender"] = client.DeviceID
			if ok := hub.SendToDevice(targetPeerID, incoming); !ok {
				log.Printf("WS Signaling Target Not Connected | type=%s sender=%s target=%s", msgType, client.DeviceID, targetPeerID)
			}

		default:
			log.Printf("WS Unknown Message Type: %s", msgType)
		}
	}
}
