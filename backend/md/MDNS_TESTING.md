# mDNS Session Discovery - Testing Guide

## How It Works

Your 0Xnet application now supports **LAN-wide session discovery** using mDNS (Multicast DNS). This means:

✅ **Devices on the SAME LAN can discover each other automatically**
✅ **Sessions created on one device are visible to all devices on the same LAN**
✅ **Devices on DIFFERENT LANs cannot see each other** (network isolation)

## Architecture

1. **mDNS Advertisement**: Each device broadcasts itself on the LAN using the service name `_0xnet._tcp`
2. **Device Discovery**: Every 10 seconds, each device scans for other 0Xnet devices on the LAN
3. **Session Aggregation**: When listing sessions, the device fetches sessions from all discovered devices

## API Endpoints

### Create Session (POST)
```bash
curl -X POST http://localhost:8080/session/create \
  -H "Content-Type: application/json" \
  -d '{"name":"My Session"}'
```

### List All Sessions (GET)
Lists sessions from the local device AND all discovered devices on the LAN
```bash
curl http://localhost:8080/session/list
```

### List Discovered Devices (GET)
Shows all 0Xnet devices found on the LAN
```bash
curl http://localhost:8080/devices
```

## Testing on the Same LAN

### Setup
1. Make sure all test devices are connected to the **same WiFi network**
2. Ensure port 8080 is not blocked by firewalls

### Test with Multiple Terminals (Single Machine)

**Terminal 1** - First device on port 8080:
```bash
cd /Users/ridhamshah/Desktop/my0Xnet/backend
go run ./cmd/app/main.go
```

**Terminal 2** - Second device on port 8081:
```bash
cd /Users/ridhamshah/Desktop/my0Xnet/backend
# Modify port in main.go or use environment variable
go run ./cmd/app/main.go
```

**Terminal 3** - Test the API:
```bash
# Create session on device 1
curl -X POST http://localhost:8080/session/create -H "Content-Type: application/json" -d '{"name":"Session from Device 1"}'

# Create session on device 2
curl -X POST http://localhost:8081/session/create -H "Content-Type: application/json" -d '{"name":"Session from Device 2"}'

# List sessions from device 1 (should see both)
curl http://localhost:8080/session/list

# List discovered devices
curl http://localhost:8080/devices
```

### Test with Different Machines on Same LAN

1. **Machine A**:
   ```bash
   cd /path/to/backend
   go run ./cmd/app/main.go
   ```

2. **Machine B**:
   ```bash
   cd /path/to/backend
   go run ./cmd/app/main.go
   ```

3. **From either machine**, create sessions and list them:
   ```bash
   # Create session on Machine A
   curl -X POST http://<machine-a-ip>:8080/session/create \
     -H "Content-Type: application/json" \
     -d '{"name":"Session A"}'
   
   # List all sessions from Machine B (should see sessions from both machines)
   curl http://<machine-b-ip>:8080/session/list
   ```

## Network Isolation

**Different LANs = No Discovery**
- Devices on different WiFi networks won't see each other
- Devices separated by routers won't discover each other
- This is by design for privacy and security

## Troubleshooting

### Devices not discovering each other
1. Check if both devices are on the same network
2. Verify firewall settings allow mDNS (port 5353 UDP)
3. Check application logs for discovery messages
4. Try `curl http://localhost:8080/devices` to see discovered devices

### Sessions not appearing
1. Wait 10 seconds after starting (discovery interval)
2. Check that `/session/list` endpoint returns data locally
3. Verify the discovered device's IP is reachable

### macOS Firewall
If discovery doesn't work, temporarily disable macOS firewall:
```bash
sudo /usr/libexec/ApplicationFirewall/socketfilterfw --setglobalstate off
```

## Implementation Details

- **Discovery Interval**: 10 seconds
- **HTTP Timeout**: 2 seconds per device
- **Service Name**: `_0xnet._tcp.local.`
- **Protocol**: mDNS (Bonjour/Zeroconf)
