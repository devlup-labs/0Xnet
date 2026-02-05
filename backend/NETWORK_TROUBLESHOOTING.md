# Network Discovery Troubleshooting Guide

## Overview
0Xnet uses mDNS (Multicast DNS) for automatic device discovery on your local network. If devices aren't discovering each other, this guide will help you diagnose and fix the issue.

## Quick Diagnostics

### 1. Check if Discovery is Running
Look for these log messages when starting the app:
```
✓ Starting mDNS advertisement for device <deviceID> on port <port>
✓ mDNS advertisement started successfully
✓ Starting device discovery for device <deviceID>
```

### 2. Check Discovered Devices
```bash
curl http://localhost:8080/devices | jq .
```

Expected output (when devices are discovered):
```json
[
  {
    "DeviceID": "some-uuid",
    "Address": "192.168.1.100",
    "Port": 8080
  }
]
```

## Common Issues and Solutions

### Issue 1: Empty Device List `[]`

**Possible Causes:**
- Devices are on different networks/VLANs
- Firewall blocking mDNS (port 5353 UDP)
- Network doesn't support multicast
- Devices haven't completed first discovery cycle (wait 10-15 seconds)

**Solutions:**

#### A. Verify Same Network
Both devices must be on the exact same network:
```bash
# On each device, check IP and subnet
ifconfig | grep "inet "
# or
ip addr show
```

Both devices should have IPs in the same subnet (e.g., 192.168.1.x)

#### B. Check Firewall
**macOS:**
```bash
# Temporarily disable firewall for testing
sudo /usr/libexec/ApplicationFirewall/socketfilterfw --setglobalstate off

# Re-enable after testing
sudo /usr/libexec/ApplicationFirewall/socketfilterfw --setglobalstate on
```

**Linux (Ubuntu/Debian):**
```bash
# Check firewall status
sudo ufw status

# Allow port 8080 and mDNS
sudo ufw allow 8080/tcp
sudo ufw allow 5353/udp
```

**Windows:**
- Go to Windows Defender Firewall
- Allow port 8080 TCP and port 5353 UDP
- Or temporarily disable firewall for private networks

#### C. Check mDNS Service
**macOS:** mDNS is built-in (Bonjour)

**Linux:**
```bash
# Install Avahi (mDNS implementation)
sudo apt-get install avahi-daemon
sudo systemctl start avahi-daemon
sudo systemctl enable avahi-daemon
```

**Windows:**
- Download and install Bonjour Print Services from Apple
- Or use iTunes which includes Bonjour

### Issue 2: Devices Discovered But Sessions Not Showing

**Symptoms:**
- `/devices` endpoint shows discovered devices
- `/session/list` returns empty array or only local sessions

**Possible Causes:**
- HTTP connection failing between devices
- Port mismatch
- Firewall blocking HTTP requests

**Solutions:**

#### A. Test Direct HTTP Connection
```bash
# From Device A, try to reach Device B
curl http://<device-b-ip>:8080/session/list

# If this fails, it's a network/firewall issue, not mDNS
```

#### B. Verify Ports Match
Check that the port in `/devices` matches the actual running port:
```bash
# Check what's actually listening
netstat -an | grep LISTEN | grep 8080
# or
lsof -i :8080
```

### Issue 3: "Can't Assign Requested Address" Error

**Symptoms:**
```
Failed to fetch sessions: dial tcp 172.31.52.30:8080: connect: can't assign requested address
```

**Cause:** The device is trying to connect to an IP that doesn't exist or isn't routable.

**Solutions:**

#### A. For Same-Machine Testing
When running multiple instances on the same machine, use localhost:
```bash
# Device 1
PORT=8080 ./app

# Device 2  
PORT=8081 ./app

# Test
curl http://localhost:8080/devices
curl http://localhost:8081/devices
```

The app will automatically try localhost for same-machine connections.

#### B. For Different Machines
Ensure both machines can ping each other:
```bash
# From Device A
ping <device-b-ip>

# From Device B
ping <device-a-ip>
```

If ping fails, check:
- Network connectivity
- Subnet masks
- Router configuration
- VPN interference

### Issue 4: Works Locally But Not Across Machines

**Possible Causes:**
- Network isolation (guest network, VLANs)
- Corporate network restrictions
- Different subnets
- AP (Access Point) isolation enabled

**Solutions:**

#### A. Check for AP Isolation
Some routers have "AP Isolation" or "Client Isolation" enabled which prevents devices from communicating:
- Log into your router settings
- Look for "AP Isolation", "Client Isolation", or "Private Mode"
- Disable this feature
- Reconnect devices

#### B. Avoid Guest Networks
Guest networks typically isolate devices. Connect all devices to the main network.

#### C. Check VLAN Configuration
In enterprise networks, devices might be on different VLANs:
```bash
# Check VLAN (Linux)
ip -d link show

# Ask your network administrator if devices are on different VLANs
```

### Issue 5: Discovery Works But Stops After Some Time

**Possible Causes:**
- Network switching (devices moving between APs)
- IP address changes (DHCP renewal)
- mDNS cache issues

**Solutions:**

#### A. Use Static IP Addresses
Configure static IPs for your test devices:
```bash
# Check current IP
ifconfig en0  # macOS
ip addr show  # Linux

# Configure static IP in your OS network settings
```

#### B. Restart Discovery
The app automatically rediscovers every 10 seconds, but you can restart the app to force immediate rediscovery.

## Testing Checklist

Use this checklist to verify your setup:

- [ ] Both devices are powered on and running 0Xnet
- [ ] Both devices show "mDNS advertisement started successfully"
- [ ] Both devices are connected to the same WiFi network
- [ ] Both devices have IPs in the same subnet (e.g., 192.168.1.x)
- [ ] You've waited at least 15 seconds after starting both apps
- [ ] `curl http://localhost:8080/devices` shows discovered devices
- [ ] Firewall allows port 8080 (TCP) and 5353 (UDP)
- [ ] You can ping from one device to the other
- [ ] You can curl `http://<other-device-ip>:8080/session/list`

## Advanced Debugging

### Enable Verbose mDNS Logging
The app already includes detailed logging. Look for these messages:

**Successful Discovery:**
```
Starting mDNS device discovery...
✓ Discovered device: <deviceID> at <ip>:<port>
Discovery complete. Found X device(s)
Current discovered devices:
  - <deviceID> at <ip>:<port>
```

**Problems:**
```
Failed to initialize resolver: <error>
Failed to browse: <error>
Ignoring incomplete entry: deviceID=<id>, addrs=[]
```

### Test mDNS Directly
**macOS/Linux:**
```bash
# Install dns-sd tool (usually pre-installed on macOS)
dns-sd -B _0xnet._tcp

# You should see services being advertised
```

**Alternative (using avahi):**
```bash
# Linux
avahi-browse -a _0xnet._tcp --resolve

# Should show all 0Xnet devices
```

### Network Packet Capture
If nothing else works, capture mDNS packets:
```bash
# macOS/Linux
sudo tcpdump -i any port 5353

# Should see mDNS queries and responses
```

## Still Having Issues?

If you've tried everything and devices still aren't discovering each other:

1. **Verify your network topology:**
   - Are devices truly on the same Layer 2 network?
   - Is multicast enabled on switches?
   - Are there any firewalls between devices?

2. **Try a different network:**
   - Create a mobile hotspot and connect both devices to it
   - This eliminates complex network infrastructure

3. **Test with same-machine setup:**
   - Run both instances on the same computer with different ports
   - If this doesn't work, there's a software issue
   - If this works but cross-machine doesn't, it's a network issue

4. **Check logs carefully:**
   - The app provides detailed logging
   - Look for error messages about resolvers, browsing, or connections
   - Share these logs when asking for help

## Network Requirements Summary

For 0Xnet device discovery to work:
- ✅ Same subnet (Layer 2 network)
- ✅ Multicast enabled
- ✅ mDNS/Bonjour service running
- ✅ Port 5353 UDP open (mDNS)
- ✅ Port 8080 TCP open (HTTP API)
- ✅ No AP isolation
- ❌ No VPNs interfering with local traffic
- ❌ No restrictive corporate firewalls
- ❌ No network segmentation/VLANs
