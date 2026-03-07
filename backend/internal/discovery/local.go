package discovery

import (
	"sync"

	"github.com/libp2p/go-libp2p/core/peer"
)

// NewLocalSessionDiscovery returns a lightweight SessionDiscovery that doesn't
// rely on libp2p. It only provides GetDiscoveredDevices for the HTTP server.
func NewLocalSessionDiscovery(localID string) *SessionDiscovery {
	sd := &SessionDiscovery{}
	sd.localDeviceID = localID
	sd.devices = make(map[peer.ID]*DiscoveredDevice)
	sd.localDevices = make(map[string]*DiscoveredDevice)
	sd.mutex = sync.RWMutex{}
	return sd
}
