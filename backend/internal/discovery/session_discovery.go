package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/bhawani-prajapat2006/0Xnet/backend/internal/models"
	"github.com/grandcat/zeroconf"
)

// DiscoveredDevice represents a device found on the LAN
type DiscoveredDevice struct {
	DeviceID string
	Address  string
	Port     int
}

// SessionDiscovery manages session discovery across the LAN
type SessionDiscovery struct {
	localDeviceID string
	devices       map[string]*DiscoveredDevice
	mutex         sync.RWMutex
}

func NewSessionDiscovery(deviceID string) *SessionDiscovery {
	return &SessionDiscovery{
		localDeviceID: deviceID,
		devices:       make(map[string]*DiscoveredDevice),
	}
}

// StartDiscovery continuously discovers devices on the LAN
func (sd *SessionDiscovery) StartDiscovery() {
	go func() {
		for {
			sd.discoverDevices()
			time.Sleep(10 * time.Second) // Rediscover every 10 seconds
		}
	}()
}

// discoverDevices finds all 0Xnet devices on the LAN using mDNS
func (sd *SessionDiscovery) discoverDevices() {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		log.Println("Failed to initialize resolver:", err)
		return
	}

	entries := make(chan *zeroconf.ServiceEntry)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go func() {
		for entry := range entries {
			// Extract device ID from TXT records
			deviceID := ""
			for _, txt := range entry.Text {
				if len(txt) > 9 && txt[:9] == "deviceId=" {
					deviceID = txt[9:]
					break
				}
			}

			// Don't add ourselves
			if deviceID == sd.localDeviceID {
				continue
			}

			if deviceID != "" && len(entry.AddrIPv4) > 0 {
				sd.mutex.Lock()
				sd.devices[deviceID] = &DiscoveredDevice{
					DeviceID: deviceID,
					Address:  entry.AddrIPv4[0].String(),
					Port:     entry.Port,
				}
				sd.mutex.Unlock()
				log.Printf("Discovered device: %s at %s:%d\n", deviceID, entry.AddrIPv4[0], entry.Port)
			}
		}
	}()

	err = resolver.Browse(ctx, "_0xnet._tcp", "local.", entries)
	if err != nil {
		log.Println("Failed to browse:", err)
	}

	<-ctx.Done()
}

// GetAllSessions fetches sessions from all discovered devices
func (sd *SessionDiscovery) GetAllSessions(localSessions []models.Session) []models.Session {
	allSessions := make([]models.Session, 0)
	
	// Add local sessions
	allSessions = append(allSessions, localSessions...)

	sd.mutex.RLock()
	devices := make([]*DiscoveredDevice, 0, len(sd.devices))
	for _, device := range sd.devices {
		devices = append(devices, device)
	}
	sd.mutex.RUnlock()

	// Fetch sessions from each discovered device
	for _, device := range devices {
		sessions := sd.fetchSessionsFromDevice(device)
		allSessions = append(allSessions, sessions...)
	}

	return allSessions
}

// fetchSessionsFromDevice fetches sessions from a specific device
func (sd *SessionDiscovery) fetchSessionsFromDevice(device *DiscoveredDevice) []models.Session {
	url := fmt.Sprintf("http://%s:%d/session/list", device.Address, device.Port)
	
	client := &http.Client{
		Timeout: 2 * time.Second,
	}
	
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("Failed to fetch sessions from %s: %v\n", device.DeviceID, err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	var sessions []models.Session
	if err := json.Unmarshal(body, &sessions); err != nil {
		return nil
	}

	return sessions
}

// GetDiscoveredDevices returns the list of discovered devices
func (sd *SessionDiscovery) GetDiscoveredDevices() []*DiscoveredDevice {
	sd.mutex.RLock()
	defer sd.mutex.RUnlock()

	devices := make([]*DiscoveredDevice, 0, len(sd.devices))
	for _, device := range sd.devices {
		devices = append(devices, device)
	}
	return devices
}
