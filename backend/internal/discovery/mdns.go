package discovery

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/grandcat/zeroconf"
)

// MDNSDevice represents a device discovered via mDNS on the LAN.
type MDNSDevice struct {
	Host     string
	Port     int
	DeviceID string
}

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

// FindAllDevices browses for ALL _0xnet._tcp services on the LAN and returns
// a slice of every device found (not just the first one).
func FindAllDevices() []MDNSDevice {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		log.Printf("mDNS resolver error: %v", err)
		return nil
	}

	entries := make(chan *zeroconf.ServiceEntry)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var devices []MDNSDevice

	go func() {
		for e := range entries {
			var ip string
			if len(e.AddrIPv4) > 0 {
				ip = e.AddrIPv4[0].String()
			} else if len(e.AddrIPv6) > 0 {
				ip = e.AddrIPv6[0].String()
			} else {
				continue
			}

			// Extract deviceId from TXT records
			devID := ""
			for _, txt := range e.Text {
				if len(txt) > 9 && txt[:9] == "deviceId=" {
					devID = txt[9:]
				}
			}
			if devID == "" {
				devID = fmt.Sprintf("%s:%d", ip, e.Port)
			}

			devices = append(devices, MDNSDevice{
				Host:     ip,
				Port:     e.Port,
				DeviceID: devID,
			})
		}
	}()

	if err := resolver.Browse(ctx, "_0xnet._tcp", "local.", entries); err != nil {
		log.Printf("mDNS browse error: %v", err)
		return nil
	}

	<-ctx.Done()
	return devices
}

// StartMDNSDiscoveryLoop runs a background loop that periodically scans for
// mDNS devices and registers/unregisters them in the SessionDiscovery so they
// appear in the /devices and /session/list endpoints.
func StartMDNSDiscoveryLoop(ctx context.Context, sd *SessionDiscovery) {
	selfID := sd.GetLocalDeviceID()
	log.Println("🔍 Starting mDNS discovery loop...")

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	// Run an initial scan immediately
	scanAndUpdate(sd, selfID)

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping mDNS discovery loop")
			return
		case <-ticker.C:
			scanAndUpdate(sd, selfID)
		}
	}
}

func scanAndUpdate(sd *SessionDiscovery, selfID string) {
	devices := FindAllDevices()
	log.Printf("🔍 mDNS scan found %d device(s)", len(devices))

	// Build a set of currently discovered mDNS device IDs
	found := make(map[string]bool)
	for _, d := range devices {
		// Skip self
		if d.DeviceID == selfID {
			continue
		}
		mdnsID := "mdns-" + d.DeviceID
		found[mdnsID] = true
		sd.RegisterDeviceWithAddr(mdnsID, d.Host, d.Port)
	}

	// Remove stale mDNS devices that are no longer advertised
	for _, existing := range sd.GetDiscoveredDevices() {
		if len(existing.DeviceID) > 5 && existing.DeviceID[:5] == "mdns-" {
			if !found[existing.DeviceID] {
				sd.UnregisterDevice(existing.DeviceID)
			}
		}
	}
}
