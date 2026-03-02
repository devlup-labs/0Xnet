# 🚀 Network Deployment Guide — Access from Any Device

## ✅ What's Ready

Your backend server is now **fully accessible from any device on your LAN** (laptop, phone, tablet, etc).

---

## 📍 Your Device IP Address

When you run the server, it **automatically detects and displays** your local IP:

```
🌐 Local IP: 172.31.70.132
```

This IP is **stable on your LAN** and can be used by other devices to connect.

---

## 🔌 Access URLs

### Relay Mode (First Device)
```
📡 WebSocket Relay:  ws://172.31.70.132:9090/relay-ws
🌍 HTTP API:         http://172.31.70.132:8080
📊 Device List:      http://172.31.70.132:8080/devices
```

### Client Mode (Other Devices)
```
🌍 HTTP API:         http://172.31.70.132:8080
📊 Device List:      http://172.31.70.132:8080/devices
```

---

## 🔥 Quick Start

### Step 1: Start Relay (On Laptop)
```bash
cd /home/sakti/Devlup_labs_projects/0Xnet/backend
./0xnet-relay
```

**Output:**
```
╔════════════════════════════════════════╗
║  🚀 0Xnet RELAY MODE ACTIVATED        ║
╚════════════════════════════════════════╝
📱 Device ID: 4396fdd6-390...
🌐 Local IP: 172.31.70.132
📡 Relay WS: ws://172.31.70.132:9090/relay-ws
🌍 API URL:  http://172.31.70.132:8080

📱 Access from other devices:
   → http://172.31.70.132:8080/devices
```

### Step 2: Access from Your Device

**From any device on the same WiFi:**

```bash
# List all connected devices
curl http://172.31.70.132:8080/devices

# Create a session
curl -X POST http://172.31.70.132:8080/session/create \
  -H "Content-Type: application/json" \
  -d '{"name":"My Session"}'

# List all sessions
curl http://172.31.70.132:8080/session/list
```

**In Browser:**
```
http://172.31.70.132:8080/devices
```

---

## 🍎 Access from iPhone

### Option 1: Using Browser
1. Open Safari
2. Go to: `http://172.31.70.132:8080/devices`

### Option 2: Using curl (with Termux or SSH)
```bash
curl http://172.31.70.132:8080/devices
```

---

## 🤖 Access from Android Phone

### Option 1: Using Browser
1. Open Chrome
2. Go to: `http://172.31.70.132:8080/devices`

### Option 2: Using Termux (Terminal)
```bash
# Install Termux from Play Store
# Then run:
curl http://172.31.70.132:8080/devices
```

### Option 3: Using Go on Android
Copy the binary to phone and run:
```bash
./0xnet-relay
```

---

## 📡 How It Works

```
Your Laptop (Relay)
│
├─ Detects local IP: 172.31.70.132
├─ Binds to 0.0.0.0 (all interfaces)
├─ Advertises via mDNS on port 9090
└─ Exposes API on port 8080

Your Phone (Client) — Same WiFi
│
├─ Discovers relay via mDNS
├─ Connects to: ws://172.31.70.132:9090/relay-ws
└─ Accesses API: http://172.31.70.132:8080
```

---

## 🔓 Firewall (If Needed)

If you can't access from another device:

```bash
# Check if ports are open
netstat -tlnp | grep -E "8080|9090"

# Allow through firewall (Linux)
sudo ufw allow 8080
sudo ufw allow 9090

# Or disable firewall temporarily (test only)
sudo ufw disable
```

---

## 📲 Multiple Devices Test

**Terminal 1: Relay**
```bash
./0xnet-relay
```
Output: `🚀 0Xnet RELAY MODE ACTIVATED`

**Terminal 2 (same laptop): Client**
```bash
PORT=8081 go run ./cmd/app
```
Output: `🔗 0Xnet CLIENT MODE ACTIVATED`

**Terminal 3: Test**
```bash
# Both running and connected
curl http://localhost:8080/devices
curl http://localhost:8081/devices
```

---

## 🚀 Production Deployment

### On Linux Server
```bash
# Build
cd /home/sakti/Devlup_labs_projects/0Xnet/backend
go build -o 0xnet-relay ./cmd/app

# Run in background
nohup ./0xnet-relay > server.log 2>&1 &

# View logs
tail -f server.log

# Access from anywhere on network
curl http://<server-ip>:8080/devices
```

### On Raspberry Pi / ARM Device
```bash
# Build for ARM
CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build -o 0xnet-relay ./cmd/app

# Transfer and run
scp 0xnet-relay pi@raspberrypi.local:~/
ssh pi@raspberrypi.local
./0xnet-relay
```

---

## 🔧 Configuration

### Change API Port
```bash
PORT=9000 ./0xnet-relay
```

### Change Relay Port
Edit `cmd/app/main.go` — search for `:9090` and change the port.

---

## ✨ Features

✅ Auto-detects local IP  
✅ Displays network URL on startup  
✅ Binds to all interfaces (0.0.0.0)  
✅ Works offline (no internet needed)  
✅ Auto relay election via mDNS  
✅ Multi-device support  
✅ Cross-platform (Linux, Mac, Windows, ARM)  

---

## 📝 API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/devices` | GET | List all devices |
| `/session/create` | POST | Create session |
| `/session/list` | GET | List sessions |
| `/session/delete` | POST | Delete session |
| `/ws` | WebSocket | Real-time communication |
| `/relay-ws` | WebSocket | Internal relay protocol |

---

## 🐛 Troubleshooting

### Can't access from phone?
1. Ensure phone is on **same WiFi network**
2. Check laptop IP: `ifconfig | grep "inet "`
3. Try pinging laptop from phone: `ping 172.31.70.132`
4. Disable firewall: `sudo ufw disable`

### "Address already in use" error?
```bash
pkill -f 0xnet-relay
sleep 2
./0xnet-relay
```

### Relay not being discovered?
```bash
# Check mDNS (Linux/Mac)
dns-sd -B _0xnet._tcp local

# Or use avahi
avahi-browse -a | grep 0Xnet
```

---

## 🎯 Next Steps

- [x] Relay auto-election ✅
- [x] Network IP detection ✅
- [x] Multi-device access ✅
- [ ] Add relay persistence
- [ ] Add device heartbeat/health check
- [ ] Add P2P direct connection (WebRTC)

