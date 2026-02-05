package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/bhawani-prajapat2006/0Xnet/backend/internal/db"
	"github.com/bhawani-prajapat2006/0Xnet/backend/internal/discovery"
	httpapi "github.com/bhawani-prajapat2006/0Xnet/backend/internal/http"
	"github.com/bhawani-prajapat2006/0Xnet/backend/internal/service"

	"github.com/joho/godotenv"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/libp2p/go-libp2p/p2p/transport/websocket"
	"github.com/multiformats/go-multiaddr"
)

func main() {
	godotenv.Load()

	dbConn, err := db.Connect()
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}

	relayAddrStr := os.Getenv("RELAY_ADDR")
	if relayAddrStr == "" {
		log.Fatal("RELAY_ADDR not found in .env")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h, err := initLibp2pHost(ctx, relayAddrStr)
	if err != nil {
		log.Fatal("Failed to initialize libp2p: ", err)
	}
	defer h.Close()

	deviceID := h.ID().String()
	log.Printf("🚀 Starting 0Xnet | PeerID: %s", deviceID)

	sessionDiscovery := discovery.NewSessionDiscovery(h)
	sessionDiscovery.StartDiscovery()

	h.SetStreamHandler("/0xnet/session-sync/1.0.0", func(s network.Stream) {
		log.Printf("📥 Incoming session sync request from: %s", s.Conn().RemotePeer())
		localSessions, _ := service.ListSessions(dbConn)
		discovery.HandleIncomingSessionRequest(s, localSessions)
	})

	port := 8080
	if pStr := os.Getenv("PORT"); pStr != "" {
		if p, err := strconv.Atoi(pStr); err == nil {
			port = p
		}
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
		os.Exit(0)
	}()

	server := httpapi.NewServer(dbConn, deviceID, sessionDiscovery, port)
	server.Start()
}

func initLibp2pHost(ctx context.Context, relayAddr string) (host.Host, error) {
	h, err := libp2p.New(
		libp2p.ChainOptions(
			libp2p.Transport(tcp.NewTCPTransport),
			libp2p.Transport(websocket.New),
		),
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
		libp2p.EnableRelay(),
	)
	if err != nil {
		return nil, err
	}

	ma, _ := multiaddr.NewMultiaddr(relayAddr)
	relayInfo, _ := peer.AddrInfoFromP2pAddr(ma)

	if err := h.Connect(ctx, *relayInfo); err != nil {
		return nil, fmt.Errorf("relay connection failed: %w", err)
	}

	log.Println("⌛ Connection established. Stabilizing for 5 seconds...")
	time.Sleep(5 * time.Second)

	var reservation *client.Reservation
	for i := 1; i <= 5; i++ {
		reservation, err = client.Reserve(ctx, h, *relayInfo)
		if err == nil {
			log.Printf("✅ SUCCESS! Reserved slot until: %s", reservation.Expiration)
			return h, nil
		}
		log.Printf("⚠️ Attempt %d: Reservation refused (%v). Retrying in 5s...", i, err)
		time.Sleep(5 * time.Second)
	}

	return nil, fmt.Errorf("relay reservation failed: %w", err)
}