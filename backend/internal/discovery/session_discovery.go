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
	// Initialize DHT in Server mode so we can participate in the DHT
	// This is necessary for peers to discover each other
	kademliaDHT, err := dht.New(context.Background(), h, dht.Mode(dht.ModeServer))
	if err != nil {
		log.Printf("❌ Failed to create DHT: %v", err)
		// Fallback to client mode if server fails
		kademliaDHT, err = dht.New(context.Background(), h, dht.Mode(dht.ModeClient))
		if err != nil {
			log.Printf("❌ Failed to create DHT in client mode: %v", err)
		}
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

	log.Println("🔍 Starting DHT Discovery...")

	// 1. Bootstrap the DHT - connect to existing peers in the network
	if err := sd.dht.Bootstrap(ctx); err != nil {
		log.Printf("⚠️ DHT Bootstrap error: %v", err)
	} else {
		log.Println("✅ DHT Bootstrap initiated")
	}

	// Wait for DHT to be ready
	time.Sleep(2 * time.Second)

	// 2. Set up network notifier to track connections
	sd.host.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(n network.Network, conn network.Conn) {
			pID := conn.RemotePeer()
			if pID == sd.host.ID() {
				return
			}
			
			// Check if this is a relay server (skip adding relays as devices)
			protocols, _ := sd.host.Peerstore().GetProtocols(pID)
			isRelay := false
			for _, proto := range protocols {
				if proto == "/libp2p/circuit/relay/0.2.0/hop" {
					isRelay = true
					break
				}
			}
			
			if !isRelay {
				sd.mutex.Lock()
				if _, exists := sd.devices[pID]; !exists {
					sd.devices[pID] = &DiscoveredDevice{
						DeviceID: pID.String(),
					}
					log.Printf("✅ Peer Connected: %s", pID.String()[:12])
				}
				sd.mutex.Unlock()
			}
		},
		DisconnectedF: func(n network.Network, conn network.Conn) {
			pID := conn.RemotePeer()
			sd.mutex.Lock()
			delete(sd.devices, pID)
			sd.mutex.Unlock()
			log.Printf("❌ Peer Disconnected: %s", pID.String()[:12])
		},
	})
	// 3. Continuously advertise and discover
	go func() {
		routingDiscovery := routing.NewRoutingDiscovery(sd.dht)
		
		// Initial advertisement
		log.Println("📢 Initial advertisement...")
		util.Advertise(ctx, routingDiscovery, "0xnet-global-v1")
		
		// Continuously advertise presence (every 60 seconds as per DHT spec)
		go func() {
			ticker := time.NewTicker(60 * time.Second)
			defer ticker.Stop()
			
			for {
				<-ticker.C
				util.Advertise(ctx, routingDiscovery, "0xnet-global-v1")
				log.Println("📢 Re-advertising presence...")
			}
		}()

		// Initial peer search
		time.Sleep(3 * time.Second)
		
		// Continuously search for peers (more frequently at start)
		searchInterval := 10 * time.Second
		ticker := time.NewTicker(searchInterval)
		defer ticker.Stop()
		
		searchCount := 0
		for {
			searchCount++
			log.Printf("🔎 Searching for peers (attempt #%d)...", searchCount)
			
			peerChan, err := routingDiscovery.FindPeers(ctx, "0xnet-global-v1")
			if err != nil {
				log.Printf("⚠️ Discovery error: %v", err)
				<-ticker.C
				continue
			}

			found := 0
			for p := range peerChan {
				if p.ID == sd.host.ID() || p.ID == "" {
					continue
				}

				found++
				log.Printf("🔍 Found peer: %s with %d addrs", p.ID.String()[:12], len(p.Addrs))
				
				// Try to connect through the relay
				if err := sd.host.Connect(ctx, p); err != nil {
					log.Printf("⚠️ Failed to connect to %s: %v", p.ID.String()[:12], err)
				} else {
					log.Printf("✨ Successfully connected to peer: %s", p.ID.String()[:12])
				}
			}
			
			if found == 0 {
				log.Println("⏳ No peers found yet, continuing search...")
			} else {
				log.Printf("✅ Found %d peer(s) in this search", found)
			}
			
			// After a few searches, slow down the interval
			if searchCount == 5 {
				ticker.Stop()
				searchInterval = 30 * time.Second
				ticker = time.NewTicker(searchInterval)
				log.Println("🕐 Reducing search frequency to 30s intervals")
			}
			
			<-ticker.C
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