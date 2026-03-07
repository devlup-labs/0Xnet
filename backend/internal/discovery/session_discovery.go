package discovery

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/bhawani-prajapat2006/0Xnet/backend/internal/models"
)

// DiscoveredDevice represents a device found on the LAN
type DiscoveredDevice struct {
	DeviceID string `json:"device_id"`
	Address  string `json:"address"`
	Port     int    `json:"port"`
}

// SessionDiscovery manages device and session discovery across the LAN
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

// GetLocalDeviceID returns this device's own ID
func (sd *SessionDiscovery) GetLocalDeviceID() string {
	return sd.localDeviceID
}

// RegisterDevice adds a discovered device to the registry
func (sd *SessionDiscovery) RegisterDevice(id, address string, port int) {
	sd.mutex.Lock()
	defer sd.mutex.Unlock()
	sd.devices[id] = &DiscoveredDevice{
		DeviceID: id,
		Address:  address,
		Port:     port,
	}
}

// UnregisterDevice removes a device from the registry
func (sd *SessionDiscovery) UnregisterDevice(id string) {
	sd.mutex.Lock()
	defer sd.mutex.Unlock()
	delete(sd.devices, id)
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

// GetAllSessions fetches sessions from all discovered devices + local sessions
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

// fetchSessionsFromDevice fetches sessions from a specific device via HTTP
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
