package main

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

// webrtcAPI is shared across all peers; created once with default codecs.
var webrtcAPI *webrtc.API

func init() {
	m := &webrtc.MediaEngine{}
	// Register VP8, VP9, H264 (video) + Opus (audio) by default.
	if err := m.RegisterDefaultCodecs(); err != nil {
		panic(err)
	}
	webrtcAPI = webrtc.NewAPI(webrtc.WithMediaEngine(m))
}

// Peer represents one participant in the SFU.
// It owns a single WebRTC PeerConnection (to the SFU, not to other peers)
// and a WebSocket connection used for signaling.
type Peer struct {
	id   string
	room *Room

	// WebRTC PeerConnection — the media pipe to the browser.
	pc *webrtc.PeerConnection

	// senders maps TrackLocalStaticRTP → RTPSender so we can remove tracks.
	senders   map[*webrtc.TrackLocalStaticRTP]*webrtc.RTPSender
	sendersMu sync.Mutex

	// WebSocket — protected by writeMu (WriteJSON is not goroutine-safe).
	conn    *websocket.Conn
	writeMu sync.Mutex

	// ICE candidates that arrived before SetRemoteDescription.
	pendingICE   []webrtc.ICECandidateInit
	pendingMu    sync.Mutex
	remoteDescOK bool
}

// NewPeer creates a PeerConnection, wires up all Pion callbacks, and
// returns the ready-to-use Peer. Call room.AddPeer and peer.ReadLoop after.
func NewPeer(room *Room, conn *websocket.Conn) (*Peer, error) {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
			// Add TURN here for production NAT traversal:
			// {
			//   URLs:       []string{"turn:your-turn-server:3478"},
			//   Username:   "user",
			//   Credential: "pass",
			// },
		},
	}

	pc, err := webrtcAPI.NewPeerConnection(config)
	if err != nil {
		return nil, err
	}

	p := &Peer{
		id:      uuid.New().String(),
		room:    room,
		pc:      pc,
		conn:    conn,
		senders: make(map[*webrtc.TrackLocalStaticRTP]*webrtc.RTPSender),
	}

	// ── Trickle ICE ──────────────────────────────────────────────────────────
	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return // gathering complete
		}
		p.writeJSON(Message{
			Type:    MsgICECandidate,
			Payload: c.ToJSON(),
		})
	})

	// ── Connection lifecycle ──────────────────────────────────────────────────
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("[peer %s] connection state → %s", p.id, state)
		switch state {
		case webrtc.PeerConnectionStateFailed,
			webrtc.PeerConnectionStateClosed,
			webrtc.PeerConnectionStateDisconnected:
			p.close()
		}
	})

	// ── Incoming media tracks from the browser ────────────────────────────────
	// When the browser sends audio or video, Pion fires OnTrack.
	// We create a TrackLocalStaticRTP and forward raw RTP packets to it.
	// The room then distributes this forwarding track to every subscriber.
	pc.OnTrack(func(remote *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		log.Printf("[peer %s] publishing track: kind=%s codec=%s",
			p.id, remote.Kind(), remote.Codec().MimeType)

		local, err := webrtc.NewTrackLocalStaticRTP(
			remote.Codec().RTPCodecCapability,
			remote.ID(),
			remote.StreamID(),
		)
		if err != nil {
			log.Printf("[peer %s] create local track: %v", p.id, err)
			return
		}

		// Register in room — all other peers will Subscribe.
		room.PublishTrack(local, p.id)

		// Forward RTP packets from the publisher into the local track.
		// This goroutine runs for the lifetime of the track.
		go forwardRTP(p.id, remote, local)
	})

	return p, nil
}

// ── Public API ────────────────────────────────────────────────────────────────

// Subscribe adds a remote track to this peer's PeerConnection and
// triggers an SDP renegotiation so the browser starts receiving it.
func (p *Peer) Subscribe(track *webrtc.TrackLocalStaticRTP) {
	sender, err := p.pc.AddTrack(track)
	if err != nil {
		log.Printf("[peer %s] AddTrack: %v", p.id, err)
		return
	}

	p.sendersMu.Lock()
	p.senders[track] = sender
	p.sendersMu.Unlock()

	// Drain RTCP from the sender (required by Pion to keep the pipe healthy).
	go drainRTCP(p.id, sender)

	p.Renegotiate()
}

// Unsubscribe removes a track from this peer's PeerConnection.
// Call Renegotiate() afterwards to apply the change.
func (p *Peer) Unsubscribe(track *webrtc.TrackLocalStaticRTP) {
	p.sendersMu.Lock()
	sender, ok := p.senders[track]
	if ok {
		delete(p.senders, track)
	}
	p.sendersMu.Unlock()

	if !ok {
		return
	}
	if err := p.pc.RemoveTrack(sender); err != nil {
		log.Printf("[peer %s] RemoveTrack: %v", p.id, err)
	}
}

// Renegotiate creates a new SDP offer and sends it to the browser.
// The browser must respond with an "answer" message.
func (p *Peer) Renegotiate() {
	offer, err := p.pc.CreateOffer(nil)
	if err != nil {
		log.Printf("[peer %s] CreateOffer: %v", p.id, err)
		return
	}
	if err := p.pc.SetLocalDescription(offer); err != nil {
		log.Printf("[peer %s] SetLocalDescription (offer): %v", p.id, err)
		return
	}
	p.writeJSON(Message{Type: MsgRenegotiate, Payload: offer})
}

// ReadLoop is the main WebSocket receive loop — runs until the client disconnects.
func (p *Peer) ReadLoop() {
	defer p.close()

	for {
		_, raw, err := p.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
			) {
				log.Printf("[peer %s] read: %v", p.id, err)
			}
			return
		}

		var msg Message
		if err := json.Unmarshal(raw, &msg); err != nil {
			log.Printf("[peer %s] bad json: %v", p.id, err)
			continue
		}

		switch msg.Type {

		// ── Initial offer from the browser ────────────────────────────────────
		case MsgOffer:
			var sdp webrtc.SessionDescription
			if err := remarshal(msg.Payload, &sdp); err != nil {
				log.Printf("[peer %s] bad offer payload: %v", p.id, err)
				continue
			}
			p.handleOffer(sdp)

		// ── Answer from the browser (response to our renegotiate) ─────────────
		case MsgAnswer:
			var sdp webrtc.SessionDescription
			if err := remarshal(msg.Payload, &sdp); err != nil {
				log.Printf("[peer %s] bad answer payload: %v", p.id, err)
				continue
			}
			p.handleAnswer(sdp)

		// ── Trickle ICE candidate from the browser ────────────────────────────
		case MsgICECandidate:
			var c webrtc.ICECandidateInit
			if err := remarshal(msg.Payload, &c); err != nil {
				log.Printf("[peer %s] bad ICE payload: %v", p.id, err)
				continue
			}
			p.handleICECandidate(c)

		default:
			log.Printf("[peer %s] unknown message type: %q", p.id, msg.Type)
		}
	}
}

// ── Internal handlers ─────────────────────────────────────────────────────────

// handleOffer processes the browser's initial SDP offer:
//  1. SetRemoteDescription
//  2. Add all existing room tracks so the peer immediately receives them
//  3. CreateAnswer + SetLocalDescription
//  4. Send the answer back
func (p *Peer) handleOffer(sdp webrtc.SessionDescription) {
	if err := p.pc.SetRemoteDescription(sdp); err != nil {
		log.Printf("[peer %s] SetRemoteDescription (offer): %v", p.id, err)
		return
	}

	// Subscribe to tracks that were already published before this peer joined.
	for _, track := range p.room.ExistingTracks(p.id) {
		sender, err := p.pc.AddTrack(track)
		if err != nil {
			log.Printf("[peer %s] AddTrack (existing): %v", p.id, err)
			continue
		}
		p.sendersMu.Lock()
		p.senders[track] = sender
		p.sendersMu.Unlock()
		go drainRTCP(p.id, sender)
	}

	answer, err := p.pc.CreateAnswer(nil)
	if err != nil {
		log.Printf("[peer %s] CreateAnswer: %v", p.id, err)
		return
	}
	if err := p.pc.SetLocalDescription(answer); err != nil {
		log.Printf("[peer %s] SetLocalDescription (answer): %v", p.id, err)
		return
	}

	p.writeJSON(Message{Type: MsgAnswer, Payload: answer})

	// Flush any ICE candidates that arrived before the remote desc was set.
	p.pendingMu.Lock()
	p.remoteDescOK = true
	pending := p.pendingICE
	p.pendingICE = nil
	p.pendingMu.Unlock()

	for _, c := range pending {
		if err := p.pc.AddICECandidate(c); err != nil {
			log.Printf("[peer %s] AddICECandidate (flush): %v", p.id, err)
		}
	}
}

// handleAnswer processes the browser's answer to a server-initiated renegotiation.
func (p *Peer) handleAnswer(sdp webrtc.SessionDescription) {
	if err := p.pc.SetRemoteDescription(sdp); err != nil {
		log.Printf("[peer %s] SetRemoteDescription (answer): %v", p.id, err)
		return
	}

	p.pendingMu.Lock()
	p.remoteDescOK = true
	pending := p.pendingICE
	p.pendingICE = nil
	p.pendingMu.Unlock()

	for _, c := range pending {
		if err := p.pc.AddICECandidate(c); err != nil {
			log.Printf("[peer %s] AddICECandidate (flush): %v", p.id, err)
		}
	}
}

// handleICECandidate queues the candidate if no remote description is set yet,
// otherwise adds it immediately.
func (p *Peer) handleICECandidate(c webrtc.ICECandidateInit) {
	p.pendingMu.Lock()
	defer p.pendingMu.Unlock()

	if !p.remoteDescOK {
		p.pendingICE = append(p.pendingICE, c)
		return
	}
	if err := p.pc.AddICECandidate(c); err != nil {
		log.Printf("[peer %s] AddICECandidate: %v", p.id, err)
	}
}

func (p *Peer) close() {
	p.room.RemovePeer(p)
	p.pc.Close()
}

func (p *Peer) writeJSON(v interface{}) {
	p.writeMu.Lock()
	defer p.writeMu.Unlock()
	if err := p.conn.WriteJSON(v); err != nil {
		log.Printf("[peer %s] writeJSON: %v", p.id, err)
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// forwardRTP copies raw RTP packets from a remote (publisher) track into a
// local (forwarding) track. Runs in its own goroutine per incoming track.
func forwardRTP(peerID string, remote *webrtc.TrackRemote, local *webrtc.TrackLocalStaticRTP) {
	buf := make([]byte, 1500) // max RTP packet size
	for {
		n, _, err := remote.Read(buf)
		if err != nil {
			log.Printf("[peer %s] track read end: %v", peerID, err)
			return
		}
		if _, err := local.Write(buf[:n]); err != nil {
			return
		}
	}
}

// drainRTCP reads and discards RTCP packets from an RTPSender.
// Pion requires this goroutine to exist; ignoring RTCP causes backpressure.
func drainRTCP(peerID string, sender *webrtc.RTPSender) {
	buf := make([]byte, 1500)
	for {
		if _, _, err := sender.Read(buf); err != nil {
			return
		}
	}
}

// remarshal round-trips through JSON to convert interface{} → concrete type.
func remarshal(src interface{}, dst interface{}) error {
	b, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dst)
}
