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
	if port <= 0 {
		// Invalid or placeholder port, ignore registration
		return
	}
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

	// Add remote sessions
	allSessions = append(allSessions, sd.GetRemoteSessions()...)

	return allSessions
}

// GetRemoteSessions fetches sessions only from discovered remote devices
func (sd *SessionDiscovery) GetRemoteSessions() []models.Session {
	sd.mutex.RLock()
	devices := make([]*DiscoveredDevice, 0, len(sd.devices))
	for _, device := range sd.devices {
		devices = append(devices, device)
	}
	sd.mutex.RUnlock()

	log.Printf("📡 GetRemoteSessions: found %d discovered device(s) to query", len(devices))

	remoteSessions := make([]models.Session, 0)
	for _, device := range devices {
		log.Printf("📡 Querying device: %s (addr=%s, port=%d)", device.DeviceID, device.Address, device.Port)
		sessions := sd.fetchSessionsFromDevice(device)
		log.Printf("📡 Got %d session(s) from %s", len(sessions), device.DeviceID)
		remoteSessions = append(remoteSessions, sessions...)
	}

	log.Printf("📡 Total remote sessions: %d", len(remoteSessions))
	return remoteSessions
}

// fetchSessionsFromDevice fetches sessions from a specific device via HTTP
// and filters out stale sessions by checking the remote device's current ID
func (sd *SessionDiscovery) fetchSessionsFromDevice(device *DiscoveredDevice) []models.Session {
	// Skip devices with no valid port (e.g. browser clients registered with port 0)
	if device.Port <= 0 {
		return nil
	}

	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	// First, get the remote device's current deviceID via /whoami
	remoteDeviceID := sd.fetchRemoteDeviceID(client, device)

	// Fetch sessions
	url := fmt.Sprintf("http://%s:%d/session/list?source=local", device.Address, device.Port)
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

	// If we know the remote device's current ID, filter out stale sessions
	if remoteDeviceID != "" {
		filtered := make([]models.Session, 0, len(sessions))
		for _, s := range sessions {
			if s.HostID == remoteDeviceID {
				filtered = append(filtered, s)
			}
		}
		return filtered
	}

	return sessions
}

// fetchRemoteDeviceID gets the current deviceID from a remote device via /whoami
func (sd *SessionDiscovery) fetchRemoteDeviceID(client *http.Client, device *DiscoveredDevice) string {
	url := fmt.Sprintf("http://%s:%d/whoami", device.Address, device.Port)
	resp, err := client.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var result struct {
		DeviceID string `json:"deviceId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ""
	}
	return result.DeviceID
}
