package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/bhawani-prajapat2006/0Xnet/backend/internal/db"
	"github.com/bhawani-prajapat2006/0Xnet/backend/internal/discovery"
	httpapi "github.com/bhawani-prajapat2006/0Xnet/backend/internal/http"
	"github.com/bhawani-prajapat2006/0Xnet/backend/internal/identity"
	"github.com/bhawani-prajapat2006/0Xnet/backend/internal/relay"
	"github.com/bhawani-prajapat2006/0Xnet/backend/internal/transport"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

// getLocalIP returns the device's local IP address on the network
func getLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "localhost"
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

func main() {
	godotenv.Load()

	dbConn, err := db.Connect()
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}

	deviceID := identity.NewDeviceID()
	localIP := getLocalIP()

	port := 8080
	if pStr := os.Getenv("PORT"); pStr != "" {
		if p, err := strconv.Atoi(pStr); err == nil {
			port = p
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Auto-discover relay via mDNS
	host, relayPort := discovery.FindRelay()

	var sessionDiscovery = discovery.NewLocalSessionDiscovery(deviceID)

	if host == "" {
		// No relay found -> become relay
		log.Println("╔════════════════════════════════════════╗")
		log.Println("║  🚀 0Xnet RELAY MODE ACTIVATED        ║")
		log.Println("╚════════════════════════════════════════╝")
		log.Printf("📱 Device ID: %s", deviceID[:12]+"...")
		log.Printf("🌐 Local IP: %s", localIP)
		log.Printf("📡 Relay WS: ws://%s:9090/relay-ws", localIP)
		log.Printf("🌍 API URL:  http://%s:%d", localIP, port)
		log.Println("")

		hub := relay.NewHub()

		go discovery.Advertise(ctx, 9090, deviceID)

		http.HandleFunc("/relay-ws", func(w http.ResponseWriter, r *http.Request) {
			conn, _ := transport.Upgrader.Upgrade(w, r, nil)

			_, msg, _ := conn.ReadMessage()
			id := string(msg)

			hub.Register(id, conn)

			log.Printf("✅ Device joined: %s", id[:12]+"...")

			for {
				_, data, err := conn.ReadMessage()
				if err != nil {
					hub.Remove(id)
					log.Printf("❌ Device left: %s", id[:12]+"...")
					break
				}
				hub.Broadcast(id, data)
			}
		})

		// Start API using local discovery
		go func() {
			server := httpapi.NewServer(dbConn, deviceID, sessionDiscovery, port)
			server.Start()
		}()

		log.Println("✨ Relay running on :9090")
		log.Println("✨ API running on :" + fmt.Sprint(port))
		log.Println("")
		log.Println("📱 Access from other devices:")
		log.Printf("   → http://%s:%d/devices", localIP, port)
		log.Println("")

		// Blocking: run relay http for websocket relay
		if err := http.ListenAndServe(":9090", nil); err != nil {
			log.Fatal(err)
		}

	} else {
		// Relay found -> act as client
		log.Println("╔════════════════════════════════════════╗")
		log.Println("║  🔗 0Xnet CLIENT MODE ACTIVATED       ║")
		log.Println("╚════════════════════════════════════════╝")
		log.Printf("📱 Device ID: %s", deviceID[:12]+"...")
		log.Printf("🌐 Local IP: %s", localIP)
		log.Printf("📡 Connecting to relay: ws://%s:%d/relay-ws", host, relayPort)
		log.Println("")

		url := fmt.Sprintf("ws://%s:%d/relay-ws", host, relayPort)

		var conn *websocket.Conn
		for i := 1; i <= 10; i++ {
			connDial, _, err := websocket.DefaultDialer.Dial(url, nil)
			if err == nil {
				conn = connDial
				break
			}
			log.Printf("⏳ Attempt %d: Retrying relay connection...", i)
			time.Sleep(2 * time.Second)
		}

		if conn == nil {
			log.Fatal("❌ Failed to connect to relay after 10 attempts")
		}

		// Send deviceID as first message
		conn.WriteMessage(websocket.TextMessage, []byte(deviceID))

		log.Println("✅ Connected to relay!")
		log.Printf("🌍 API URL:  http://%s:%d", localIP, port)
		log.Println("")
		log.Println("📱 Access from other devices:")
		log.Printf("   → http://%s:%d/devices", localIP, port)
		log.Println("")

		// Start API using local discovery (limited)
		go func() {
			server := httpapi.NewServer(dbConn, deviceID, sessionDiscovery, port)
			server.Start()
		}()

		// Keep reading messages from relay (application-specific handling can be added)
		go func() {
			for {
				_, msg, err := conn.ReadMessage()
				if err != nil {
					log.Println("❌ Relay connection lost:", err)
					os.Exit(1)
				}
				log.Println("📨 Relay message:", string(msg))
			}
		}()

		// Keep main alive until signal
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		cancel()
		os.Exit(0)
	}
}
