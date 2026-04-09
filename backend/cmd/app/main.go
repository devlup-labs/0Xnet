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
	"github.com/bhawani-prajapat2006/0Xnet/backend/internal/streaming"

	"github.com/google/uuid"
)

// getLocalIP returns the device's local IP address on the network.
// Works completely offline by enumerating network interfaces instead
// of dialing an external server.
func getLocalIP() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "localhost"
	}

	for _, iface := range interfaces {
		// Skip loopback, down, and point-to-point interfaces
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipnet.IP.To4()
			if ip == nil {
				continue // skip IPv6
			}
			// Return the first private IPv4 address
			if ip[0] == 10 ||
				(ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31) ||
				(ip[0] == 192 && ip[1] == 168) {
				return ip.String()
			}
		}
	}

	return "localhost"
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

	// Initialize stream manager
	streamMgr := streaming.NewStreamManager()

	// Start the HTTP API server
	go func() {
		server := httpapi.NewServer(dbConn, deviceID, sessionDiscovery, port, streamMgr)
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
	streamMgr.StopAll() // Kill any active ffmpeg processes
	log.Println("Shutting down...")
}
