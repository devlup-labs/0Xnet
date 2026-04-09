# Complete Networking & Discovery Mechanics in 0Xnet

The **0Xnet** application uses a completely offline, localized approach to discover and connect with other instances of the app on the same network. Because it doesn't rely on central servers, STUN/TURN servers, or the internet, it must mathematically and programmatically determine who else is nearby, using a variety of networking protocols and techniques.

This document serves as a comprehensive glossary of **"everything"** networking-related happening inside the 0Xnet localized architecture.

---

## 1. Fundamentals

*   **IP Address (IPv4):** A unique numerical identifier assigned to every device connected to a network (e.g., `192.168.1.5`). 0Xnet exclusively uses IPv4 to simplify address mapping on local networks.
*   **Private IP Address:** IP addresses explicitly reserved for use inside Local Area Networks (LANs) that are not publicly routable on the global internet (e.g., `192.168.x.x`, `10.x.x.x`, `172.16.x.x`). 0Xnet enforces filtering to *only* accept these, ensuring military-grade offline separation.
*   **Loopback Address (Localhost):** The IP address `127.0.0.1`. It points back to your own machine. When defining network interfaces, 0Xnet ignores the loopback interface because connecting to yourself doesn’t help you find other people on the network.
*   **Port:** If an IP Address is a street address, the Port is the apartment number. 0Xnet defaults to port `8080`.
*   **Network Interface:** The physical or virtual adapter your computer uses to connect to a network (like your Wi-Fi card or Ethernet adapter). 0Xnet loops through all hardware interfaces to find the one actively broadcasting a Private IPv4 address.

## 2. Advanced Subnetting

*   **Subnet (Subnetwork):** A logical subdivision of a larger network. Your router places local devices into a specific subnet (like a contained neighborhood) so they can communicate directly without leaving the LAN.
*   **Subnet Mask:** A mathematical 32-bit mask (e.g., `255.255.255.0`) that splits an IP address into a "Network Address" (the street) and a "Host Address" (the house number).
*   **Network Address:** The very first IP in a subnet (`192.168.1.0`), reserved to identify the network itself. 0Xnet drops this address during a sweep.
*   **Broadcast Address:** The very last IP in a subnet (`192.168.1.255`), normally used to send packets to *everyone* simultaneously via UDP. 0Xnet removes this during its sweep because it specifically connects TCP/HTTP connections to individual hosts.
*   **CIDR Notation (Classless Inter-Domain Routing):** Shorthand notation pairing the IP and mask (e.g., `192.168.1.5/24`). A `/24` mask means there are exactly 254 available IPs to sweep for peers.

## 3. The 0Xnet Subnet Sweep Algorithm

Instead of relying on mDNS (Multicast DNS) or UDP broadcasting—which are frequently blocked by enterprise firewalls or modern Windows Network profiles—0Xnet relies on a **Subnet Sweep**, also known as a Ping Sweep or TCP Probe.

1.  **CIDR Math:** Using Golang's `net` package, the app takes the CIDR block (e.g., `/24`) and generates a massive slice/array containing all 254 possible IP strings (`.1`, `.2`, `.3` ... `.254`).
2.  **Concurrency (Goroutines):** A "Goroutine" is Go's version of a super lightweight thread. 0Xnet spawns 254 simultaneous goroutines to check every single IP address at the exact same time.
3.  **Semaphore Control:** To prevent crashing the operating system by opening too many network sockets at once, a "Semaphore" (a buffered channel of size 100) controls the traffic, ensuring only 100 goroutines probe at any given microsecond.
4.  **The Probe (HTTP Request):** Each goroutine fires a 500-millisecond HTTP GET request to `http://<target_ip>:8080/whoami`. 
5.  **Validation:** If someone is running `python -m http.server` on port 8080, it will respond, but it's not 0Xnet. The backend reads the response body expecting a strict JSON payload `{ "deviceId": "..." }`. If valid, it records the device as a legitimate peer.

## 4. Peer-To-Peer Communication Mechanics

Once discovery finishes, the actual application communication begins via multiple differing protocols:

*   **HTTP (REST API):** Used for single, stateless actions. Fetching available sessions, fetching metadata, or posting a "Join Request" is done purely through standard, one-off HTTP requests.
*   **WebSockets (WS):** An upgraded HTTP connection that stays open permanently in both directions. This forms the backbone of 0Xnet. The "Host" runs a WebSocket Server Hub, and all "Guests" connect their clients to it. When someone sends a chat message, it goes up the active WebSocket port, and the host bounces it asynchronously to everyone else.
*   **WebRTC (Web Real-Time Communication):** The engine used for the video and audio calls. WebRTC sets up a true **Peer-to-Peer Mesh** connections. Video traffic doesn't route through the Host; it flows directly from Guest A's computer to Guest B's computer (Point-to-Point).
*   **Signaling:** Before WebRTC can establish a direct video connection, the two computers must exchange setup information (SDP offers/answers and ICE candidates). 0Xnet uses the existing **WebSocket** backbone passing through the Host machine as the "Signaling Server".
*   **FFmpeg & HLS (HTTP Live Streaming):** Used for Media Sharing (like movie files). Because raw video files are heavy, the Host's backend uses the FFmpeg engine to split movies into tiny 3-second `.ts` chunks on the fly. The Guests then pull these chunks natively over the local network via HLS `m3u8` playlists, creating Netflix-style adaptive playback without pre-loading an entire 2GB file.
