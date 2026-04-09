# HTTP Live Streaming (HLS) in 0Xnet

The `backend/internal/streaming` package powers the "Share Media" feature in 0Xnet. 

While WebRTC is perfect for live webcams (`MediaStream`), it is historically poor at broadcasting large 4K movie files or highly compressed pre-recorded media across a peer-to-peer network without dropping frames or losing sync. Thus, 0Xnet uses **HLS (HTTP Live Streaming)** locally.

---

## 1. The Architecture

When the Host selects a video file (MP4, MKV, AVI) from their computer:
1.  **Local Instant Play**: The host's frontend immediately uses a local `blob://` URL. They don't wait for transcoding, so playback starts instantly on their machine.
2.  **Transcode Engine (FFmpeg)**: In the background, the file is shipped to the Go backend via `/stream/upload`. The backend spawns a raw `ffmpeg` instance directly on the host's operating system.
3.  **Adaptive Chunking**: FFmpeg rapidly chops the video into small 2-second `.ts` chunks and documents them in an `index.m3u8` playlist file.
4.  **Guest Retrieval**: The Host broadcasts a WebSocket `stream-started` event. The Guests receive the URL to the `m3u8` playlist and their client (`hls.js` inside React) begins pulling down the 2-second chunks natively over the local IP.

## 2. Dynamic FFmpeg Strategies (`ffmpeg.go`)

Because 0Xnet emphasizes zero loading times, the `buildFFmpegArgs` logic makes intelligent decisions based on the input file type.

### A. The "Remux" Fast Path (MP4 / MOV)
```go
args = append(args, "-c", "copy")
```
If the host uploads an `.mp4` or `.mov` file, the backend assumes the video is already compressed beautifully (likely H.264 + AAC). FFmpeg completely skips "transcoding" and performs a "Remux". It simply splits the file into pieces without altering the pixels. This operates at massive speeds (often 50x real-time), entirely eliminating buffering for guests.

### B. The Transcode Brute Force Path (MKV / AVI / WebM)
```go
args = append(args, "-c:v", "libx264", "-preset", "ultrafast")
```
If the file format isn't web-native, FFmpeg kicks into overdrive using the CPU. It transcodes the video to `H.264` and the audio to `AAC` using the `ultrafast` preset, guaranteeing that everyone on the session can watch the video regardless of device compatibility, while sacrificing a tiny margin of CPU overhead.

### C. Zero-Latency Tuning
To prevent viewers from waiting 10 seconds for the engine to warm up, the arguments contain:
*   `-analyzeduration 2000000`: Limits input probing to 2 seconds instead of reading the whole header.
*   `-hls_time 2`: Keeps chunks microscopically small (2 seconds).
*   `-hls_init_time 0`: Forces FFmpeg to spit out the very first chunk the millisecond it's built, meaning the `index.m3u8` file goes live almost instantly.

## 3. Streaming Sync (The WebSocket Relay)

HLS natively provides high-quality buffing, but it doesn't solve "Watch Parties" out of the box. 

If the Host clicks "Pause" or scrubs to a new scene:
1.  The Host's React player fires a `'play'`, `'pause'`, or `'seeked'` event.
2.  This sends a JSON WebSocket payload with `type: "sync-playback"`.
3.  The backend's `SessionHub` catches this and uses `Broadcast()` to relay it to all connected Guests cleanly.
4.  The Guests' React players catch the message, compare their current `<video>.currentTime`, and programmatically skip or pause to explicitly mirror the Host's exact frame!
