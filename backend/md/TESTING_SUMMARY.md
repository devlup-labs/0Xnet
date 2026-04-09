# 0Xnet Multi-Device Testing — Quick Start

## Status ✅ (Server Running)

Both relay and client are **running successfully** on same machine:

```
Relay:  http://localhost:8080  (PID 53365)
Client: http://localhost:8081  (PID 53932)
```

Both have databases initialized and are serving API requests.

---

## Test on Your Mobile Phone (Same WiFi)

### Prerequisites
- **Mobile phone** on the same WiFi as your laptop
- **Go installed** (optional; can use the binary instead)

### Option 1: Using Go (Recommended for Testing)

1. **On Laptop**, find your local IP:
```bash
ifconfig | grep "inet " | grep -v 127.0.0.1
# Example output: 192.168.1.100
```

2. **Transfer code to phone** (via USB, email, GitHub):
```bash
# On your laptop
cd /home/sakti/Devlup_labs_projects/0Xnet
# Zip and send to phone
zip -r 0Xnet-backend.zip backend/
```

3. **On mobile phone** (with Termux or Linux environment):
```bash
# Extract the code
unzip 0Xnet-backend.zip
cd backend

# Make sure phone WiFi is on same network as laptop
# Then run:
go run ./cmd/app
```

**Expected Output:**
```
Found relay at: 192.168.x.x:9090
Connected to relay as device-id-xxxx
🌍 0Xnet API active on port 8080
```

### Option 2: Using Binary (Faster)

1. **Copy binary to phone:**
```bash
# From laptop
adb push /home/sakti/Devlup_labs_projects/0Xnet/backend/0xnet-relay /data/local/tmp/

# Or via email/USB if adb not available
```

2. **On phone (Termux):**
```bash
/data/local/tmp/0xnet-relay
```

---

## Verify Connection From Phone

On your phone (in Termux or browser):

```bash
# Test API
curl http://localhost:8080/devices

# Should return:
[{"device_id":"xxxx (Me)"}]
```

---

## Current Test Results (Laptop Only)

**Relay (port 8080):**
```json
[{"device_id":"7851854b-2087-4ae4-984e-cbab6627c1a7 (Me)"}]
```

**Client (port 8081):**
```json
[{"device_id":"0c8be781-c9e1-478f-b35c-cf10fcf5bc20 (Me)"}]
```

Both running, but `GetDiscoveredDevices()` not populating cross-device list yet.  
**Next:** Implement relay → client device registry sync.

---

## Kill Running Instances

```bash
# Kill relay
pkill -f "0xnet-relay"

# Kill client (go run)
pkill -f "go run ./cmd/app"

# Verify
ps aux | grep -E "0xnet|go run" | grep -v grep
```

---

## Network Debugging

If phone cannot find relay:

1. **Ping relay from phone:**
```bash
# On phone, ping your laptop's IP
ping 192.168.1.100
```

2. **Check mDNS on laptop:**
```bash
# Install avahi-tools if needed
sudo apt-get install avahi-utils

# Browse mDNS services
avahi-browse -a | grep 0Xnet
```

3. **If mDNS not found:**
   - Ensure both devices on **same WiFi** (not guest/5GHz mismatch)
   - Disable firewall temporarily: `sudo ufw disable`
   - Check router: some routers block mDNS across VLANs

---

## Next Steps

- [ ] Test on mobile phone with binary
- [ ] Verify relay announces via mDNS
- [ ] Implement device list sync across relay → clients
- [ ] Add relay failover (if relay dies, next device becomes relay)

---

## Files

- **Relay binary:** `/home/sakti/Devlup_labs_projects/0Xnet/backend/0xnet-relay` (20MB)
- **Full testing guide:** `MULTI_DEVICE_TESTING.md`
- **Logs:** `/tmp/relay.log`, `/tmp/client.log`

---

## Quick Commands

```bash
# Start relay
cd /home/sakti/Devlup_labs_projects/0Xnet/backend
./0xnet-relay > /tmp/relay.log 2>&1 &

# Start client (different port)
PORT=8081 go run ./cmd/app > /tmp/client.log 2>&1 &

# Check both running
curl http://localhost:8080/devices
curl http://localhost:8081/devices

# View logs
tail -f /tmp/relay.log
tail -f /tmp/client.log

# Kill all
pkill -f "0xnet-relay"; pkill -f "go run ./cmd/app"
```

---

## What's Working ✅

- [x] Relay auto-election (first device becomes relay)
- [x] mDNS advertisement on port 9090
- [x] Client auto-discovery via mDNS
- [x] WebSocket relay connection (/relay-ws)
- [x] HTTP API on separate port
- [x] Database initialization
- [x] Multi-instance on same machine (using PORT env var)

## What's Next 🔄

- [ ] Cross-relay device discovery sync
- [ ] Relay failover / re-election
- [ ] P2P encryption (optional)

