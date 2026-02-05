package discovery

import (
	"context"
	"log"

	"github.com/grandcat/zeroconf"
)

func Advertise(ctx context.Context, port int, deviceID string) {
	// Register the mDNS service with more detailed logging
	log.Printf("Starting mDNS advertisement for device %s on port %d", deviceID, port)
	
	server, err := zeroconf.Register(
		"0Xnet-"+deviceID,
		"_0xnet._tcp",
		"local.",
		port,
		[]string{"deviceId=" + deviceID},
		nil,
	)
	if err != nil {
		log.Printf("Failed to start mDNS advertisement: %v", err)
		return
	}
	defer server.Shutdown()
	
	log.Printf("mDNS advertisement started successfully for device %s", deviceID)

	// Keep advertising until context is cancelled
	<-ctx.Done()
	log.Println("Stopping mDNS advertisement")
}
