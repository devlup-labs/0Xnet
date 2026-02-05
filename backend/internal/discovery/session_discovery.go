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
	relayPeer     peer.ID
}

func NewSessionDiscovery(h host.Host, relayPeerID peer.ID) *SessionDiscovery {
	kademliaDHT, err := dht.New(context.Background(), h, dht.Mode(dht.ModeClient))
	if err != nil {
		log.Printf("❌ Failed to create DHT: %v", err)
	}

	return &SessionDiscovery{
		host:          h,
		localDeviceID: h.ID().String(),
		devices:       make(map[peer.ID]*DiscoveredDevice),
		dht:           kademliaDHT,
		relayPeer:     relayPeerID,
	}
}

func (sd *SessionDiscovery) StartDiscovery() {
	ctx := context.Background()

	// 1. Seed the DHT with the relay
	if sd.relayPeer != "" {
		if err := sd.dht.ConnectBootstrap(ctx, sd.relayPeer); err != nil {
			log.Printf("⚠️ DHT Seeding failed: %v", err)
		}
	}

	if err := sd.dht.Bootstrap(ctx); err != nil {
		log.Printf("⚠️ DHT Bootstrap error: %v", err)
	}

	// 2. Track real-time connections via Notifier
	sd.host.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(n network.Network, conn network.Conn) {
			pID := conn.RemotePeer()
			if pID == sd.host.ID() || pID == sd.relayPeer {
				return
			}
			sd.mutex.Lock()
			sd.devices[pID] = &DiscoveredDevice{DeviceID: pID.String()}
			sd.mutex.Unlock()
			log.Printf("✅ Peer Connected & Tracked: %s", pID.String()[:12])
		},
		DisconnectedF: func(n network.Network, conn network.Conn) {
			pID := conn.RemotePeer()
			sd.mutex.Lock()
			delete(sd.devices, pID)
			sd.mutex.Unlock()
			log.Printf("❌ Peer Disconnected: %s", pID.String()[:12])
		},
	})

	// 3. Background Discovery Loop
	go func() {
		routingDiscovery := routing.NewRoutingDiscovery(sd.dht)
		
		// Advertising
		go func() {
			for {
				util.Advertise(ctx, routingDiscovery, "0xnet-global-v1")
				time.Sleep(30 * time.Second)
			}
		}()

		// Searching
		for {
			peerChan, err := routingDiscovery.FindPeers(ctx, "0xnet-global-v1")
			if err == nil {
				for p := range peerChan {
					if p.ID == sd.host.ID() || p.ID == "" || p.ID == sd.relayPeer {
						continue
					}
					sd.host.Connect(ctx, p)
				}
			}
			time.Sleep(20 * time.Second)
		}
	}()
}

// --- API & SYNC FUNCTIONS ---

// GetDiscoveredDevices returns the list for the /devices endpoint
func (sd *SessionDiscovery) GetDiscoveredDevices() []*DiscoveredDevice {
	sd.mutex.RLock()
	defer sd.mutex.RUnlock()
	list := make([]*DiscoveredDevice, 0, len(sd.devices))
	for _, d := range sd.devices {
		list = append(list, d)
	}
	return list
}

// HandleIncomingSessionRequest is the responder for when OTHER peers call you
func HandleIncomingSessionRequest(s network.Stream, sessions []models.Session) {
	defer s.Close()
	if err := json.NewEncoder(s).Encode(sessions); err != nil {
		log.Printf("Error sending sessions to peer: %v", err)
	}
}

// FetchSessionsFromPeer is the requester that YOU use to pull data from a specific peer
func (sd *SessionDiscovery) FetchSessionsFromPeer(pID peer.ID) ([]models.Session, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Open a new stream using the protocol ID we defined in main.go
	stream, err := sd.host.NewStream(ctx, pID, "/0xnet/session-sync/1.0.0")
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	var sessions []models.Session
	if err := json.NewDecoder(stream).Decode(&sessions); err != nil {
		return nil, err
	}

	return sessions, nil
}

// GetAllSessions combines local and remote data (The big picture function)
func (sd *SessionDiscovery) GetAllSessions(localSessions []models.Session) []models.Session {
	sd.mutex.RLock()
	peersToSync := make([]peer.ID, 0, len(sd.devices))
	for pID := range sd.devices {
		peersToSync = append(peersToSync, pID)
	}
	sd.mutex.RUnlock()

	allSessions := localSessions

	// Iterate through all discovered peers and pull their data
	for _, pID := range peersToSync {
		remoteSessions, err := sd.FetchSessionsFromPeer(pID)
		if err != nil {
			log.Printf("⚠️ Failed to sync with %s: %v", pID, err)
			continue
		}
		allSessions = append(allSessions, remoteSessions...)
	}

	return allSessions
}