package main

import (
	"log"
	"sync"

	"github.com/pion/webrtc/v3"
)

// publishedTrack pairs a forwarding track with the ID of its publisher
// so we can clean up when that publisher leaves.
type publishedTrack struct {
	track       *webrtc.TrackLocalStaticRTP
	publisherID string
}

// Room holds all peers and published tracks for a single call.
type Room struct {
	id      string
	onEmpty func() // called when the last peer leaves

	mu     sync.RWMutex
	peers  map[string]*Peer
	tracks []publishedTrack
}

func newRoom(id string, onEmpty func()) *Room {
	return &Room{
		id:      id,
		onEmpty: onEmpty,
		peers:   make(map[string]*Peer),
	}
}

// ── Peer management ──────────────────────────────────────────────────────────

func (r *Room) AddPeer(p *Peer) {
	r.mu.Lock()
	r.peers[p.id] = p
	r.mu.Unlock()
}

// RemovePeer removes the peer and all tracks it published, then notifies
// remaining subscribers so they can renegotiate.
func (r *Room) RemovePeer(p *Peer) {
	r.mu.Lock()

	delete(r.peers, p.id)

	// Collect and remove any tracks published by this peer.
	remaining := r.tracks[:0]
	var removedTracks []*webrtc.TrackLocalStaticRTP
	for _, pt := range r.tracks {
		if pt.publisherID == p.id {
			removedTracks = append(removedTracks, pt.track)
		} else {
			remaining = append(remaining, pt)
		}
	}
	r.tracks = remaining

	peerCount := len(r.peers)
	subscribers := make([]*Peer, 0, peerCount)
	for _, sub := range r.peers {
		subscribers = append(subscribers, sub)
	}

	r.mu.Unlock()

	// Remove departed tracks from each subscriber's PeerConnection and
	// trigger renegotiation so the client drops the dead streams.
	for _, sub := range subscribers {
		for _, t := range removedTracks {
			sub.Unsubscribe(t)
		}
		if len(removedTracks) > 0 {
			sub.Renegotiate()
		}
	}

	if peerCount == 0 && r.onEmpty != nil {
		r.onEmpty()
	}
}

// ── Track management ─────────────────────────────────────────────────────────

// PublishTrack registers a new forwarding track and subscribes all current
// peers (except the publisher) to it.
func (r *Room) PublishTrack(track *webrtc.TrackLocalStaticRTP, publisherID string) {
	r.mu.Lock()
	r.tracks = append(r.tracks, publishedTrack{track: track, publisherID: publisherID})

	subscribers := make([]*Peer, 0, len(r.peers))
	for id, p := range r.peers {
		if id != publisherID {
			subscribers = append(subscribers, p)
		}
	}
	r.mu.Unlock()

	log.Printf("[room %s] track published by %s → %d subscribers", r.id, publisherID, len(subscribers))

	for _, sub := range subscribers {
		sub.Subscribe(track)
	}
}

// ExistingTracks returns a snapshot of all currently published tracks,
// excluding those from the given peer (so a publisher doesn't receive itself).
func (r *Room) ExistingTracks(excludePeerID string) []*webrtc.TrackLocalStaticRTP {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]*webrtc.TrackLocalStaticRTP, 0, len(r.tracks))
	for _, pt := range r.tracks {
		if pt.publisherID != excludePeerID {
			out = append(out, pt.track)
		}
	}
	return out
}
