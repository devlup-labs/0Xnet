package discovery

import (
	"context"
	"log"
	"time"

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

// FindRelay searches for a relay advertised via mDNS and returns its host and port.
func FindRelay() (string, int) {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		log.Printf("mDNS resolver error: %v", err)
		return "", 0
	}

	entries := make(chan *zeroconf.ServiceEntry)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var host string
	var port int

	go func() {
		for e := range entries {
			if len(e.AddrIPv4) > 0 {
				host = e.AddrIPv4[0].String()
				port = e.Port
				return
			}
		}
	}()

	if err := resolver.Browse(ctx, "_0xnet._tcp", "local.", entries); err != nil {
		log.Printf("mDNS browse error: %v", err)
		return "", 0
	}

	<-ctx.Done()
	return host, port
}
