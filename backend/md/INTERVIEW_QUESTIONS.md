# 0Xnet Interview Questions

If you are showcasing **0Xnet** as a portfolio project in a technical interview, use these potential questions and answers to articulate the technical depth, architectural decisions, and networking complexities of the system.

---

## 1. Networking & Discovery

**Q: How does 0Xnet discover peers on the network without using an internet-based server or a centralized database?**
**A:** Historically, systems might use UDP broadcast or mDNS (Multicast DNS) for this, but these are frequently blocked by modern corporate firewalls or Windows/macOS network profiles. Instead, 0Xnet uses a "Subnet Sweep". The Go backend accesses the computer's network interface, identifies the current Private IP and its Subnet Mask (like `/24`), and calculates all 254 possible local addresses. It then concurrently fires asynchronous HTTP GET requests to port `8080` for every single IP. Whichever addresses reply with a valid 0Xnet JSON payload are officially discovered as instances of the app.

**Q: How do you prevent that massive Subnet Sweep from crashing the application or the host's operating system?**
**A:** It relies on Go's concurrent Goroutines combined with a Semaphore pattern. By creating a buffered channel with a capacity of 100, we ensure that at any given microsecond, a maximum of 100 network sockets are attempting to open. This prevents overwhelming the OS file descriptor limits and gracefully manages the HTTP probing while keeping the sweep duration down to milliseconds.

---

## 2. WebSockets & Persistent Connection

**Q: Why use a WebSocket instead of HTTP Polling for chat messages and WebRTC signaling?**
**A:** HTTP is fundamentally stateless; every request requires a heavy TCP handshake, sending headers, and receiving headers, which creates overhead and latency. By upgrading an HTTP connection to a WebSocket upon joining a session, we create a persistent, full-duplex TCP tunnel. This allows the backend to proactively push chat messages, playback synchronization, and WebRTC SDP keys to connected clients the exact millisecond they happen, completely eliminating polling latency.

**Q: In your WebSocket implementation, how do you prevent race conditions when two users join simultaneously?**
**A:** The `SessionManager` and each `SessionHub` in the Go backend encapsulate their maps of active connection pointers with an `sync.RWMutex` (Read/Write Mutex). Before inserting a new client into the map or removing a disconnected one, a Write Lock (`Lock()`) blocks other threads. When broadcasting messages, a Read Lock (`RLock()`) allows concurrent reads but blocks any thread from changing the connection map until the broadcast finishes.

---

## 3. WebRTC Architecture

**Q: What is WebRTC, and why do you use it for video but not for streaming movies?**
**A:** WebRTC (Web Real-Time Communication) establishes a true direct Peer-to-Peer network (UDP tunnel) for sub-millisecond audio and video streams between webcams. It is ideal for instantaneous live communication, but it is not built to efficiently compress and transmit pre-recorded 2GB 4K `.mp4` movie files reliably across network jitter without severe frame dropping. For movies, we shift to HLS over HTTP.

**Q: WebRTC normally requires STUN and TURN servers to work. How did you bypass that?**
**A:** STUN and TURN servers handle NAT traversal—they help computers punch holes through firewalls to communicate across the chaotic public internet. Due to 0Xnet’s localized design, all clients exist on the same private subnetwork. By explicitly passing an empty array `iceServers: []` in the `RTCPeerConnection` configuration, the WebRTC ICE engine is forced to strictly utilize "Host" candidates. It simply swaps internal LAN IPs (e.g., `192.168.1.5` to `192.168.1.10`) and connects instantly without relying on any external cloud dependencies.

**Q: What happens if two clients try to call each other at the exact same time when they join? (Glare issue)**
**A:** The frontend employs a "Polite Caller" arbitration logic. Before generating an SDP Offer, it uses JavaScript's `String.prototype.localeCompare()` to compare the device ID of the local client versus the target peer. The client that yields alphabetically first is assigned the active role to generate the Offer, while the other automatically steps back to wait for it.

---

## 4. Media Streaming (FFmpeg & HLS)

**Q: How does the application handle sharing large video files between peers instantly without buffering?**
**A:** 0Xnet implements localized HTTP Live Streaming (HLS) combined with Go's `os/exec` package executing FFmpeg. When the Host selects a file, it starts playing locally immediately via an instantaneous `URL.createObjectURL` blob. Simultaneously, FFmpeg is spawned in the background, chopping the file into tiny 2-second `.ts` chunks documented in an `index.m3u8` playlist. Guests then natively pull these miniature chunks via `hls.js` over their standard local network speeds.

**Q: FFmpeg transcoding is notorious for taking a huge amount of CPU power. How did you mitigate this?**
**A:** The Go backend intelligently analyzes the file. If the uploaded file is a modern `.mp4` or `.mov`, FFmpeg skips transcoding entirely and uses a "Remux" pipeline (`-c copy`). This simply repacks the bytes into `.ts` chunks with zero video-re-encoding happening, effectively operating at 50x real-time speed. It only falls back to heavy CPU rendering (`libx264` via the `ultrafast` preset) if the user uploads an unsupported file format like `.mkv` or `.avi`.

**Q: If you stream the movie via HLS, how do you handle syncing pauses or timeline scrubs across all viewers?**
**A:** The viewers' HLS buffers pull chunks independently, avoiding the traditional "Watch Party" desync issue. For play/pause control, 0Xnet leverages the separate persistent WebSocket connection. When the Host pauses, it pushes a `sync-playback` JSON payload. The backend's `SessionHub` instantly broadcasts this payload to the connected Guests. Their React players read the message and programmatically update their underlying `<video>.currentTime` to perfectly sync with the Host.

---

## 5. Potential System Design & Scaling Follow-Ups

**Q: If you were to scale 0Xnet from a local LAN app to a cloud-hosted app for 50 people, what architecture changes would you need to make?**
**A:** 
1. **WebRTC:** True Mesh Topologies crash browsers at around 6-8 participants. For 50 members, we must replace Mesh with a Selective Forwarding Unit (SFU) like `pion/webrtc`. Instead of 49 streams, every participant sends exactly 1 upload stream to the server, and the server blindly forwards it down to the others.
2. **Discovery:** We must remove the Subnet Sweep entirely, replacing it with a centralized Postgres/Redis database to locate users by uniquely generated Room Codes.
3. **STUN/TURN:** Because participants would be separated across the internet behind separate firewalls, we would need to deploy coturn servers to provide external ICE IPs and relay endpoints.
