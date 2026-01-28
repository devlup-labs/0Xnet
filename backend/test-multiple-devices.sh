#!/bin/bash
# Script to test multiple devices on the same machine

# Kill any existing instances
killall app 2>/dev/null

# Build the app
echo "Building app..."
go build -o app ./cmd/app

# Start Device 1 on port 8080
echo "Starting Device 1 on port 8080..."
PORT=8080 ./app &
DEVICE1_PID=$!

# Start Device 2 on port 8081
echo "Starting Device 2 on port 8081..."
PORT=8081 ./app &
DEVICE2_PID=$!

# Wait for servers to start
sleep 3

echo ""
echo "=== Both devices are running ==="
echo "Device 1: http://localhost:8080"
echo "Device 2: http://localhost:8081"
echo ""

# Test creating sessions
echo "Creating session on Device 1..."
curl -X POST http://localhost:8080/session/create \
  -H "Content-Type: application/json" \
  -d '{"name":"Device 1 Session"}' 2>/dev/null
echo ""

echo "Creating session on Device 2..."
curl -X POST http://localhost:8081/session/create \
  -H "Content-Type: application/json" \
  -d '{"name":"Device 2 Session"}' 2>/dev/null
echo ""

# Wait for discovery (10 seconds)
echo ""
echo "Waiting 12 seconds for mDNS discovery..."
sleep 12

# Check discovered devices
echo ""
echo "=== Devices discovered by Device 1 ==="
curl http://localhost:8080/devices 2>/dev/null | jq .
echo ""

echo "=== Devices discovered by Device 2 ==="
curl http://localhost:8081/devices 2>/dev/null | jq .
echo ""

# Check sessions (should see both)
echo "=== Sessions visible to Device 1 (should see both) ==="
curl http://localhost:8080/session/list 2>/dev/null | jq .
echo ""

echo "=== Sessions visible to Device 2 (should see both) ==="
curl http://localhost:8081/session/list 2>/dev/null | jq .
echo ""

echo "Press Ctrl+C to stop all devices..."
wait
