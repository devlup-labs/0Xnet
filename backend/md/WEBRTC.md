# WebRTC (Web Real-Time Communication) in 0Xnet

0Xnet utilizes **WebRTC** to power its live video and audio sharing. While the discovery and chat functions rely on standard HTTP and WebSockets, WebRTC is used exclusively for the real-time media because it can establish true P2P (Peer-to-Peer) connections.

Here is a comprehensive breakdown of how 0Xnet implements WebRTC in a strictly offline LAN environment.

---

## 1. Network Topology: Full Mesh

0Xnet currently uses a **Full Mesh Topology**.
*   **What this means:** Instead of sending video to a central server which then broadcasts it out to everyone else (an SFU/MCU model), every single participant connects directly to every single other participant.
*   If there are 4 people in a call, your computer is actively keeping 3 outbound video streams alive, and receiving 3 inbound streams simultaneously. 

## 2. No STUN and No TURN (Offline Native)

WebRTC typically struggles to connect two computers blindly across the internet because of routers and firewalls blocking direct access (known as NAT). To fix this, regular apps like Zoom use servers called STUN and TURN to punch holes through firewalls or relay video data.
*   **0Xnet Design:** Because 0Xnet is designed specifically for **local offline LANs**, it forcefully explicitly disables STUN/TURN servers.
*   **Implementation (`LiveSession.tsx`):**
    ```javascript
    const pc = new RTCPeerConnection({
      iceServers: [] // Purposely left completely empty
    })
    ```
*   **Result:** This forces WebRTC's ICE (Interactive Connectivity Establishment) engine to exclusively generate "Host" candidates—meaning it only uses your machine's true local IPs (like `192.168.1.15`), guaranteeing traffic never attempts to reach the internet.

## 3. The Signaling Process

WebRTC cannot magically discover IPs. Before two computers inside the 0Xnet session can swap direct video, they must complete a "handshake" to share codecs, resolutions, and exact IP pathways. 0Xnet uses its existing **WebSocket** backbone routing through the Host as the broker.

**The "Polite Caller" Logic:**
If two peers enter a room exactly simultaneously, they might both maliciously scream "I'm calling you!" at each other, causing the WebRTC connection state to crash. 0Xnet solves this by comparing local device IDs alphabetically.
*   `localId.localeCompare(remoteId) < 0`
*   Only the device whose ID is "first" in the alphabet takes on the role of the Caller.

**The Handshake Flow:**
1.  **Offer:** The Caller creates an SDP Offer (Session Description Protocol) detailing its camera feeds. It sends this via WebSocket to the Callee.
2.  **Answer:** The Callee receives the Offer, accepts it, and creates an SDP Answer, passing it back via the WebSocket.
3.  **ICE Candidates:** While the SDPs are flying, the background networking engine (ICE) generates candidate IPs. The app intercepts the `onicecandidate` event and tosses those IP addresses across the WebSocket too.
4.  **Connection:** Once both peers combine the SDPs and the ICE IPs, WebRTC abandons the WebSocket path entirely and snaps together a direct UDP (User Datagram Protocol) tunnel directly across your Wi-Fi or Ethernet.

## 4. Media Tracks

Once the direct tunnel opens, the `ontrack` event fires on both machines.
*   0Xnet intercepts these incoming `MediaStreamTrack` objects and saves them directly into React state (`setRemoteStreams`).
*   The UI maps over this state, spawning individual `<video>` HTML elements and directly assigning the incoming streams to bypass complex React rendering loops. 

*(Note: There is an archived internal `videoCall/` folder that experiments with a Selective Forwarding Unit (SFU) utilizing the Go `pion/webrtc` engine. It's preserved for massive scaling efforts later!)*
