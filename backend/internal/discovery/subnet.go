package discovery

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// getLocalIPAndSubnet returns the local IP address and its CIDR notation
func getLocalIPAndSubnet() (string, string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	localIP := localAddr.IP

	// Find the matching network interface to get the subnet mask
	interfaces, err := net.Interfaces()
	if err != nil {
		return localIP.String(), "", err
	}

	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				if ipnet.IP.Equal(localIP) {
					return localIP.String(), ipnet.String(), nil
				}
			}
		}
	}

	return localIP.String(), "", fmt.Errorf("could not find subnet for %s", localIP)
}

// generateSubnetIPs returns all usable host IPs in the given CIDR range
func generateSubnetIPs(cidr string) ([]string, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	var ips []string
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); incIP(ip) {
		ips = append(ips, ip.String())
	}

	// Remove network address (first) and broadcast address (last)
	if len(ips) > 2 {
		return ips[1 : len(ips)-1], nil
	}
	return ips, nil
}

// incIP increments an IP address by 1
func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// probeIP checks if a 0Xnet app is running at the given IP:port
// Returns the device ID string if found, or error if not
func probeIP(client *http.Client, ip string, port int) (string, error) {
	url := fmt.Sprintf("http://%s:%d/devices", ip, port)

	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}

	// This IP has a 0Xnet app running — use IP:port as device ID
	deviceID := fmt.Sprintf("subnet-%s:%d", ip, port)
	return deviceID, nil
}

// ScanSubnet sweeps all IPs in the local subnet to find 0Xnet devices
func ScanSubnet(port int, selfIP string) []DiscoveredDevice {
	_, cidr, err := getLocalIPAndSubnet()
	if err != nil || cidr == "" {
		// Fallback: assume /24 subnet from our own IP
		parts := strings.Split(selfIP, ".")
		if len(parts) == 4 {
			cidr = fmt.Sprintf("%s.%s.%s.0/24", parts[0], parts[1], parts[2])
		} else {
			log.Println("❌ Could not determine subnet")
			return nil
		}
	}

	ips, err := generateSubnetIPs(cidr)
	if err != nil {
		log.Printf("❌ Failed to generate subnet IPs: %v", err)
		return nil
	}

	// Fast HTTP client with short timeout for LAN probing
	client := &http.Client{
		Timeout: 500 * time.Millisecond,
	}

	var mu sync.Mutex
	var found []DiscoveredDevice

	// Limit concurrency to 100 goroutines at a time
	sem := make(chan struct{}, 100)
	var wg sync.WaitGroup

	for _, ip := range ips {
		// Skip our own IP
		if ip == selfIP {
			continue
		}

		wg.Add(1)
		go func(targetIP string) {
			defer wg.Done()
			sem <- struct{}{}        // acquire semaphore slot
			defer func() { <-sem }() // release semaphore slot

			deviceID, err := probeIP(client, targetIP, port)
			if err == nil {
				mu.Lock()
				found = append(found, DiscoveredDevice{
					DeviceID: deviceID,
					Address:  targetIP,
					Port:     port,
				})
				mu.Unlock()
			}
		}(ip)
	}

	wg.Wait()
	return found
}

// StartSubnetDiscoveryLoop runs a background loop that periodically sweeps
// the subnet to find 0Xnet devices. This replaces the old mDNS discovery.
func StartSubnetDiscoveryLoop(ctx context.Context, sd *SessionDiscovery, port int, selfIP string) {
	log.Println("🔍 Starting Subnet Sweep discovery...")
	log.Printf("🌐 Scanning from local IP: %s on port %d", selfIP, port)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Run an initial scan immediately
	sweepAndUpdate(sd, port, selfIP)

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping Subnet Sweep discovery")
			return
		case <-ticker.C:
			sweepAndUpdate(sd, port, selfIP)
		}
	}
}

// sweepAndUpdate runs one sweep cycle and updates the device registry
func sweepAndUpdate(sd *SessionDiscovery, port int, selfIP string) {
	devices := ScanSubnet(port, selfIP)

	if len(devices) > 0 {
		log.Printf("🔍 Subnet sweep found %d device(s)", len(devices))
	}

	// Track which devices were found in this sweep
	found := make(map[string]bool)
	for _, d := range devices {
		found[d.DeviceID] = true
		sd.RegisterDevice(d.DeviceID, d.Address, d.Port)
	}

	// Remove devices that didn't respond this time (went offline)
	for _, existing := range sd.GetDiscoveredDevices() {
		if strings.HasPrefix(existing.DeviceID, "subnet-") {
			if !found[existing.DeviceID] {
				sd.UnregisterDevice(existing.DeviceID)
			}
		}
	}
}
