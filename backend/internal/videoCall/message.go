package main

// Signaling message types.
const (
	// Client → Server
	MsgOffer        = "offer"         // Initial SDP offer from joining peer
	MsgAnswer       = "answer"        // SDP answer after server-initiated renegotiation
	MsgICECandidate = "ice-candidate" // Trickle ICE candidate

	// Server → Client
	MsgRenegotiate = "renegotiate" // New SDP offer from SFU (new track available)
	MsgError       = "error"       // Something went wrong
)

// Message is the WebSocket envelope for all signaling traffic.
//
// Flow (client publishes):
//  1. Client → Server : offer  (with audio/video sendrecv)
//  2. Server → Client : answer
//  3. Both sides       : ice-candidate (trickle)
//
// Flow (new track arrives for a subscriber):
//  4. Server → Client : renegotiate  (new offer with added track)
//  5. Client → Server : answer
type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload,omitempty"`
}
