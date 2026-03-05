package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/bhawani-prajapat2006/0Xnet/backend/internal/db"
	"github.com/bhawani-prajapat2006/0Xnet/backend/internal/discovery"
	httpapi "github.com/bhawani-prajapat2006/0Xnet/backend/internal/http"
	"github.com/bhawani-prajapat2006/0Xnet/backend/internal/identity"

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

	sessionDiscovery := discovery.NewLocalSessionDiscovery(deviceID)

	log.Println("╔════════════════════════════════════════╗")
	log.Println("║  🚀 0Xnet PEER MODE ACTIVATED         ║")
	log.Println("╚════════════════════════════════════════╝")
	log.Printf("📱 Device ID: %s", deviceID[:12]+"...")
	log.Printf("🌐 Local IP: %s", localIP)
	log.Printf("🌍 API URL:  http://%s:%d", localIP, port)
	log.Println("")

	// Advertise this device via mDNS
	go discovery.Advertise(ctx, port, deviceID)

	// Start continuous mDNS discovery loop to find other devices on the LAN
	go discovery.StartMDNSDiscoveryLoop(ctx, sessionDiscovery)

	// Start the HTTP API server
	go func() {
		server := httpapi.NewServer(dbConn, deviceID, sessionDiscovery, port)
		server.Start()
	}()

	log.Println("✨ API running on :" + fmt.Sprint(port))
	log.Println("")
	log.Println("📱 Access from other devices:")
	log.Printf("   → http://%s:%d/devices", localIP, port)
	log.Println("")

	// Keep main alive until signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	cancel()
	log.Println("Shutting down...")
}
