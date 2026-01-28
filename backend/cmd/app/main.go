package main

import (
	"log"
	"os"
	"strconv"

	"github.com/bhawani-prajapat2006/0Xnet/backend/internal/db"
	"github.com/bhawani-prajapat2006/0Xnet/backend/internal/discovery"
	httpapi "github.com/bhawani-prajapat2006/0Xnet/backend/internal/http"

	"github.com/google/uuid"
)

func main() {
	deviceID := uuid.New().String()
	
	// Get port from environment variable, default to 8080
	port := 8080
	if portStr := os.Getenv("PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	dbConn, err := db.Connect()
	if err != nil {
		log.Fatal(err)
	}

	// Start mDNS advertisement
	go discovery.Advertise(port, deviceID)

	// Initialize session discovery
	sessionDiscovery := discovery.NewSessionDiscovery(deviceID)
	sessionDiscovery.StartDiscovery()

	server := httpapi.NewServer(dbConn, deviceID, sessionDiscovery, port)
	log.Printf("0Xnet running on port %d\n", port)
	server.Start()
}
