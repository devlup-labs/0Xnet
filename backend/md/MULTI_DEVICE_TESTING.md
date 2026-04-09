# Multi-Device Testing Guide for 0Xnet Offline Relay

## Overview
This guide shows how to test the auto-relay election on multiple devices (laptop, phone, tablet) on the **same WiFi network**.

---

## Architecture Recap
```
WiFi / LAN Network
│
├── Device A (Laptop)  → Starts first → Becomes RELAY
├── Device B (Phone)   → Starts 2nd  → CONNECTS as Client
├── Device C (Laptop2) → Starts 3rd  → CONNECTS as Client
│
All devices = offline, no internet needed ✓
```

---

## Prerequisites

### 1. All Devices on Same WiFi
- **Laptop A**: Connected to `HomeWiFi` (192.168.x.x)
- **Phone B**: Connected to `HomeWiFi` (192.168.x.x)
- Verify with: `ifconfig` or `ipconfig`

### 2. Build the Binary (Optional but Recommended)
If you want to run on mobile without Go installed, build once on your laptop:

```bash
cd /home/sakti/Devlup_labs_projects/0Xnet/backend

# For Linux/Mac
go build -o 0xnet-relay ./cmd/app

# For Windows
go build -o 0xnet-relay.exe ./cmd/app
```

This creates a standalone binary you can copy to other devices.

---

## Test Scenario 1: Two Laptops on Same LAN

### Step 1: Start Relay (Laptop A)
```bash
# On Laptop A (192.168.1.100)
cd /home/sakti/Devlup_labs_projects/0Xnet/backend
go run ./cmd/app
```

**Expected Output:**
```
2026/02/06 13:11:40 No relay found. Becoming relay...
2026/02/06 13:11:40 Relay running on :9090
2026/02/06 13:11:40 Starting mDNS advertisement for device xxxxxxx on port 9090
2026/02/06 13:11:40 🌍 0Xnet API active on port 8080
2026/02/06 13:11:40 mDNS advertisement started successfully...
```

✅ **Relay is up and advertising via mDNS on port 9090.**

---

### Step 2: Start Client (Laptop B)
Open a **new terminal** on Laptop B (same WiFi, different device):

```bash
# On Laptop B (192.168.1.101)
cd /home/sakti/Devlup_labs_projects/0Xnet/backend
go run ./cmd/app
```

**Expected Output:**
```
2026/02/06 13:12:00 Found relay at: 192.168.1.100:9090
2026/02/06 13:12:01 Connected to relay as device-id-xyz
2026/02/06 🌍 0Xnet API active on port 8080
```

✅ **Client discovered relay and connected!**

---

### Step 3: Verify Connection via HTTP API

On **Laptop B**, test the API to see discovered devices:

```bash
curl http://localhost:8080/devices
```

**Expected Response:**
```json
[
  {
    "device_id": "device-id-relay (Me)"
  },
  {
    "device_id": "device-id-xyz"
  }
]
```

✅ **Both devices are visible to each other.**

---

## Test Scenario 2: Mobile Phone as Client

### Step 1: Build Binary (Skip if Go installed on Phone)
```bash
cd /home/sakti/Devlup_labs_projects/0Xnet/backend
go build -o 0xnet-relay ./cmd/app

# Copy to phone via USB or email
# On Android: Place in /data/local/tmp/0xnet-relay (requires adb)
# On iOS: Not directly possible without jailbreak; use Go iOS dev setup
```

### Step 2: Run on Phone
If Go is installed via Termux (Android):
```bash
cd /sdcard/0Xnet/backend
go run ./cmd/app
```

Or if using the binary:
```bash
/data/local/tmp/0xnet-relay
```

**Expected Output:**
```
Found relay at: 192.168.1.100:9090
Connected to relay as device-id-phone
🌍 0Xnet API active on port 8080
```

---

## Test Scenario 3: Three Devices (Relay + 2 Clients)

### Step 1: Laptop A - Relay
```bash
go run ./cmd/app
```
Output: `Becoming relay... Relay running on :9090`

### Step 2: Laptop B - Client 1
```bash
go run ./cmd/app
```
Output: `Found relay at: 192.168.1.100:9090`

### Step 3: Phone C - Client 2
```bash
go run ./cmd/app
```
Output: `Found relay at: 192.168.1.100:9090`

**Verify All Connected:**
On any device:
```bash
curl http://localhost:8080/devices
```

Should list all 3 devices.

---

## Debugging Issues

### Issue 1: Client Cannot Find Relay
**Cause:** mDNS discovery failing (different subnets, firewall blocking).

**Debug:**
```bash
# On client, check if relay is advertised
# Linux/Mac:
dns-sd -B _0xnet._tcp local

# Windows:
nslookup 0Xnet-<deviceid>.local

# If not found, check firewall:
sudo iptables -L -n | grep -i mdns
```

**Fix:**
- Ensure all devices on **same WiFi network** (not guest WiFi)
- Disable firewall temporarily: `sudo ufw disable` (Linux)
- Restart the relay: `Ctrl+C` then `go run ./cmd/app`

### Issue 2: Connected but Cannot Reach API
**Cause:** Port 8080 blocked by firewall.

**Debug:**
```bash
# From client, check if relay is reachable
curl http://192.168.1.100:8080/devices
```

**Fix:**
```bash
# Allow port 8080
sudo ufw allow 8080
```

### Issue 3: Relay Dies, Clients Reconnect
**Current behavior:** Client exits when relay dies.
**Future improvement:** Auto-reconnect and re-election (coming soon).

---

## Quick Test Commands

### On Relay Device:
```bash
# Start relay
go run ./cmd/app &

# Check if mDNS is advertising
dns-sd -B _0xnet._tcp local

# Test API
curl http://localhost:8080/devices
```

### On Client Device:
```bash
# Start client
go run ./cmd/app &

# Check if connected
curl http://localhost:8080/devices

# View relay messages (if any)
tail -f /tmp/0xnet.log
```

---

## Expected Behavior Summary

| State | Relay | Client 1 | Client 2 |
|-------|-------|----------|----------|
| Device 1 starts | ✅ Becomes relay | — | — |
| Device 2 starts | Running | ✅ Finds & connects | — |
| Device 3 starts | Running | Connected | ✅ Finds & connects |
| All see each other? | ✅ Yes | ✅ Yes | ✅ Yes |
| Relay dies | ❌ Offline | ❌ Disconnects | ❌ Disconnects |
| Relay restarts | ✅ New relay | ✅ Auto-reconnects | ✅ Auto-reconnects |

---

## Next Steps (Future Enhancements)

- [ ] Add re-election logic when relay dies
- [ ] Add persistent device list across reconnects
- [ ] Add WebRTC for direct P2P communication (bypass relay)
- [ ] Add encryption for relay messages

---

## Support

If tests fail, enable debug logs:
```bash
export DEBUG=true
go run ./cmd/app
```

Or open an issue with:
- Device types (laptop, phone, OS)
- Network info (`ifconfig` output)
- Log output from `go run ./cmd/app`
