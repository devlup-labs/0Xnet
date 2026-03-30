package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	// Restrict to your domain in production.
	CheckOrigin: func(r *http.Request) bool { return true },
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	sfu := NewSFU()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		roomID := r.URL.Query().Get("room")
		if roomID == "" {
			http.Error(w, `missing ?room= query parameter`, http.StatusBadRequest)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("[main] websocket upgrade: %v", err)
			return
		}

		room := sfu.GetOrCreateRoom(roomID)
		peer, err := NewPeer(room, conn)
		if err != nil {
			log.Printf("[main] create peer: %v", err)
			conn.WriteJSON(Message{Type: MsgError, Payload: err.Error()})
			conn.Close()
			return
		}

		room.AddPeer(peer)
		log.Printf("[sfu] peer %s joined room %s", peer.id, roomID)

		// Blocks until the WebSocket disconnects.
		peer.ReadLoop()

		log.Printf("[sfu] peer %s left room %s", peer.id, roomID)
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	log.Printf("🚀 WebRTC SFU listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
