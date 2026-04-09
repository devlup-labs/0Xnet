# WebSockets Implementation in 0Xnet

The `backend/internal/websocket` package provides the persistent, bidirectional communication skeleton for 0Xnet. Once Discovery is complete and users click "Join", the REST APIs give way to exactly *one* persistent WebSocket connection per guest. 

This connection is multiplexed to handle chat messages, video playback synchronization, and WebRTC signaling (SDP/ICE handshakes).

Here is a breakdown of the specific logic found in this package.

---

## 1. The Global Session Manager (`SessionManager`)

Because 0Xnet might hypothetically host multiple separate groups or "Rooms" at once, everything is federated by `sessionID`.
*   **`SessionManager` Structure:** This is essentially a giant dictionary (a `map[string]*SessionHub`) that associates a unique `sessionID` with a specific `SessionHub`.
*   **Global Instance:** There is a single `var GlobalManager = NewSessionManager()` acting as the true root of all persistent connection state across the entire backend backend runtime.
*   **Concurrency Safe:** It utilizes a `sync.RWMutex` so that multiple users connecting exactly at the same time don't corrupt the backend dictionaries. 

## 2. The Session Hub (`SessionHub`)

If the `SessionManager` is an apartment complex, the `SessionHub` is a single apartment where people inside can talk to each other.
*   **Registration Structure:** Tracks all active connections in a thread-safe map: `Clients map[*Client]bool`.
*   **`Register` / `Unregister`:** When someone authenticates, their connection struct is mapped here, allowing them to receive broadcasts. When their WebSocket drops, they are removed to prevent server memory bloat.
*   **Delivery Methods:** 
    *   `Broadcast`: Sends the exact same JSON payload to *every* client.
    *   `BroadcastExcluding`: Sends a payload to everyone *except* the creator (useful for avoiding sending a chat message to the person who just wrote it).
    *   `SendToDevice`: Traverses the client list until it finds the specific `DeviceID` requested. It uniquely routes a message strictly to that client (crucial for private WebRTC handshake operations).

## 3. The Handler Logic (`handler.go`)

This is the router logic representing the "Upgraded" HTTP endpoint on the server.

### Phase 1: Authentication (Handshake Upgrade)
When the frontend points at `ws://<host_ip>:8080/ws`, the `ServeWS` function intercepts the standard HTTP request wrapper and calls `upgrader.Upgrade(w, r, nil)`. This flips the connection from stateless HTTP into an open TCP pipe.
*   The pipeline immediately hangs, demanding an initial `join-session` JSON blob pointing out the `SessionID` and the user's `DeviceID`.
*   Once authorized, it places them in the corresponding `SessionHub`.

### Phase 2: The Infinite Message Loop (Multiplexing)
Once the setup is done, a continuous `for { ... }` block loops endlessly, listening for `conn.ReadJSON`. Because 0Xnet has numerous real-time features, it tags incoming JSON with `"type"` keys. The loop actively reads and redirects this traffic based on its `type` logic:

1. **`chat`:** 
   Simple broadcast. The backend attaches the user's name and spits it back to the `Hub.Broadcast()`, rendering on everyone's screen in milliseconds.
2. **`sync-playback`:** 
   Used heavily for the HLS movie sharing. If the host pauses the video, the pause command hits the WebSocket, and is passed verbatim via `Hub.Broadcast()` so all viewers' players pause natively in sync.
3. **WebRTC Signaling (`offer`, `answer`, `ice-candidate`, `renegotiate`):** 
   If clients were to blast video setup passwords/hashes to *everybody*, connections would break. WebRTC relies strictly on single-target point-to-point bridging. The handler detects WebRTC payloads and explicitly utilizes `Hub.SendToDevice(targetPeerId)` to deliver network traverse details natively and securely.
