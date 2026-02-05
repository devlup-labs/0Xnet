package discovery

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/bhawani-prajapat2006/0Xnet/backend/internal/models"
	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/libp2p/go-libp2p/p2p/discovery/util"
)

type DiscoveredDevice struct {
	DeviceID string `json:"device_id"`
}

type SessionDiscovery struct {
	host          host.Host
	localDeviceID string
	devices       map[peer.ID]*DiscoveredDevice
	mutex         sync.RWMutex
	dht           *dht.IpfsDHT
}

func NewSessionDiscovery(h host.Host) *SessionDiscovery {
	// Initialize DHT in Client mode (so we don't try to be a relay ourselves)
	kademliaDHT, err := dht.New(context.Background(), h, dht.Mode(dht.ModeClient))
	if err != nil {
		log.Printf("❌ Failed to create DHT: %v", err)
	}

	return &SessionDiscovery{
		host:          h,
		localDeviceID: h.ID().String(),
		devices:       make(map[peer.ID]*DiscoveredDevice),
		dht:           kademliaDHT,
	}
}

func (sd *SessionDiscovery) StartDiscovery() {
	ctx := context.Background()

	// 1. Bootstrap the DHT
	if err := sd.dht.Bootstrap(ctx); err != nil {
		log.Printf("⚠️ DHT Bootstrap error: %v", err)
	}

	routingDiscovery := routing.NewRoutingDiscovery(sd.dht)

	// 2. Advertise your presence globally
	util.Advertise(ctx, routingDiscovery, "0xnet-global-v1")
	log.Println("📢 Broadcasting presence to global network...")

	// 3. Background loop to find peers
	go func() {
		for {
			peerChan, err := routingDiscovery.FindPeers(ctx, "0xnet-global-v1")
			if err != nil {
				log.Printf("❌ Discovery error: %v", err)
				time.Sleep(10 * time.Second)
				continue
			}

			for p := range peerChan {
				if p.ID == sd.host.ID() || p.ID == "" {
					continue
				}

				sd.mutex.Lock()
				if _, exists := sd.devices[p.ID]; !exists {
					// Connect to discover their multiaddress through the relay
					if err := sd.host.Connect(ctx, p); err == nil {
						sd.devices[p.ID] = &DiscoveredDevice{
							DeviceID: p.ID.String(),
						}
						log.Printf("✨ Discovered Colleague: %s", p.ID.String()[:12])
					}
				}
				sd.mutex.Unlock()
			}
			time.Sleep(20 * time.Second)
		}
	}()
}

func (sd *SessionDiscovery) GetDiscoveredDevices() []*DiscoveredDevice {
	sd.mutex.RLock()
	defer sd.mutex.RUnlock()
	
	list := make([]*DiscoveredDevice, 0, len(sd.devices))
	for _, d := range sd.devices {
		list = append(list, d)
	}
	return list
}

func (sd *SessionDiscovery) GetAllSessions(localSessions []models.Session) []models.Session {
	// For now, just return local sessions
	// In the future, you could fetch sessions from discovered peers via libp2p streams
	return localSessions
}

// HandleIncomingSessionRequest handles incoming session sync requests from other peers
func HandleIncomingSessionRequest(s network.Stream, sessions []models.Session) {
	defer s.Close()
	if err := json.NewEncoder(s).Encode(sessions); err != nil {
		log.Printf("Error sending sessions to peer: %v", err)
	}
}