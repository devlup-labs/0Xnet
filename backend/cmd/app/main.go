/*
package main

import (

	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/bhawani-prajapat2006/0Xnet/backend/internal/db"
	"github.com/bhawani-prajapat2006/0Xnet/backend/internal/discovery"
	httpapi "github.com/bhawani-prajapat2006/0Xnet/backend/internal/http"

	"github.com/google/uuid"

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
		deviceID := uuid.New().String()
		localIP := getLocalIP()

		// Get port from environment variable, default to 8080
		port := 8080
		if portStr := os.Getenv("PORT"); portStr != "" {
			if p, err := strconv.Atoi(portStr); err == nil {
				port = p
			}
		}

		dbConn, err := db.Connect()
		if err != nil {
			log.Fatal("Database connection failed:", err)
		}

		// Initialize session discovery
		sessionDiscovery := discovery.NewSessionDiscovery(deviceID)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		log.Println("╔════════════════════════════════════════╗")
		log.Println("║  🚀 0Xnet PEER MODE ACTIVATED         ║")
		log.Println("╚════════════════════════════════════════╝")
		log.Printf("📱 Device ID: %s", deviceID[:12]+"...")
		log.Printf("🌐 Local IP: %s", localIP)
		log.Printf("🌍 API URL:  http://%s:%d", localIP, port)
		log.Println("")

		// Start Subnet Sweep discovery loop (replaces mDNS)
		go discovery.StartSubnetDiscoveryLoop(ctx, sessionDiscovery, port, localIP)

		// Start the HTTP API server
		go func() {
			server := httpapi.NewServer(dbConn, deviceID, sessionDiscovery, port)
			server.Start()
		}()

		log.Println("📱 Access from other devices:")
		log.Printf("   → http://%s:%d/devices", localIP, port)
		log.Printf("   → http://%s:%d/session/list", localIP, port)
		log.Println("")

		// Keep main alive until signal
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		cancel()
		log.Println("Shutting down...")
	}
*/
package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/bhawani-prajapat2006/0Xnet/backend/internal/db"
	"github.com/bhawani-prajapat2006/0Xnet/backend/internal/discovery"
	httpapi "github.com/bhawani-prajapat2006/0Xnet/backend/internal/http"
	"github.com/bhawani-prajapat2006/0Xnet/backend/internal/service"

	"github.com/google/uuid"
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
	deviceID := uuid.New().String()
	localIP := os.Getenv("HOST_IP")
	if localIP == "" {
		localIP = getLocalIP()
	}

	// Get port from environment variable, default to 8080
	port := 8080
	if portStr := os.Getenv("PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	dbConn, err := db.Connect()
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}

	// Clean up stale sessions from previous runs (deviceID changes on each restart)
	service.CleanupStaleSessions(dbConn, deviceID)

	// Initialize session discovery
	sessionDiscovery := discovery.NewSessionDiscovery(deviceID)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Println("╔════════════════════════════════════════╗")
	log.Println("║  🚀 0Xnet PEER MODE ACTIVATED         ║")
	log.Println("╚════════════════════════════════════════╝")
	log.Printf("📱 Device ID: %s", deviceID[:12]+"...")
	log.Printf("🌐 Local IP: %s", localIP)
	log.Printf("🌍 API URL:  http://%s:%d", localIP, port)
	log.Println("")

	// Start Subnet Sweep discovery loop (replaces mDNS)
	go discovery.StartSubnetDiscoveryLoop(ctx, sessionDiscovery, port, localIP)

	// Start the HTTP API server
	go func() {
		server := httpapi.NewServer(dbConn, deviceID, sessionDiscovery, port)
		server.Start()
	}()

	log.Println("📱 Access from other devices:")
	log.Printf("   → http://%s:%d/devices", localIP, port)
	log.Printf("   → http://%s:%d/session/list", localIP, port)
	log.Println("")

	// Keep main alive until signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	cancel()
	log.Println("Shutting down...")
}
