package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
)

// AgentInfo represents the registration data to be sent to the monitoring server.
type AgentInfo struct {
	Hostname  string   `json:"hostname"`
	IP        string   `json:"ip"`
	OpenPorts []int    `json:"openPorts"`
	Timestamp int64    `json:"timestamp"`
	AgentPort int      `json:"agentPort"`
}

// Metrics represents the system metrics to be sent.
type Metrics struct {
	Hostname  string  `json:"hostname"`
	IP        string  `json:"ip"`
	Timestamp int64   `json:"timestamp"`
	CPUUsage  float64 `json:"cpuUsage"`
	DiskUsage float64 `json:"diskUsage"`
	RAMUsage  float64 `json:"ramUsage"`
}

// getHostname retrieves the system hostname.
func getHostname() (string, error) {
	return os.Hostname()
}

// getLocalIP returns a non-loopback local IP address.
func getLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}
	return "", fmt.Errorf("cannot find local IP")
}

// parsePorts parses a comma-separated string of ports and ranges into a slice of integers.
// Example: "8080,9000-9090,1433"
func parsePorts(s string) ([]int, error) {
	var ports []int
	tokens := strings.Split(s, ",")
	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if strings.Contains(token, "-") {
			parts := strings.Split(token, "-")
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid port range: %s", token)
			}
			start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err != nil {
				return nil, err
			}
			end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err != nil {
				return nil, err
			}
			if start > end {
				return nil, fmt.Errorf("invalid port range, start > end: %s", token)
			}
			for i := start; i <= end; i++ {
				ports = append(ports, i)
			}
		} else {
			port, err := strconv.Atoi(token)
			if err != nil {
				return nil, err
			}
			ports = append(ports, port)
		}
	}
	return ports, nil
}

// getOpenPorts returns the list of ports to be included in the AgentInfo.
// If the PORTS environment variable is set, it returns exactly that list (without checking if they are open).
// Otherwise, it scans all ports (1 to 65535) and returns only those that are open.
func getOpenPorts() []int {
	portsEnv := os.Getenv("PORTS")
	if portsEnv != "" {
		p, err := parsePorts(portsEnv)
		if err != nil {
			fmt.Printf("Error parsing PORTS environment variable: %v\n", err)
			// Fallback to scanning all ports if parsing fails.
		} else {
			return p
		}
	}
	// If PORTS is not set or parsing fails, scan all ports and return only the open ones.
	var openPorts []int
	var wg sync.WaitGroup
	var mu sync.Mutex

	const startPort = 1
	const endPort = 65535
	timeout := 200 * time.Millisecond

	// Limit concurrency to 100 workers.
	sem := make(chan struct{}, 100)

	for port := startPort; port <= endPort; port++ {
		wg.Add(1)
		sem <- struct{}{} // Acquire a slot.
		go func(p int) {
			defer wg.Done()
			defer func() { <-sem }() // Release the slot.
			address := fmt.Sprintf("127.0.0.1:%d", p)
			conn, err := net.DialTimeout("tcp", address, timeout)
			if err == nil {
				mu.Lock()
				openPorts = append(openPorts, p)
				mu.Unlock()
				conn.Close()
			}
		}(port)
	}

	wg.Wait()
	return openPorts
}

// registerAgent sends the agent registration information to the monitoring server.
func registerAgent(agentInfo AgentInfo, serverURL string) error {
	jsonData, err := json.Marshal(agentInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal agent info: %v", err)
	}

	resp, err := http.Post(serverURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send registration: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registration failed with status: %s", resp.Status)
	}

	fmt.Printf("Agent registration successful: %s\n", resp.Status)
	return nil
}

// sendMetrics sends the collected system metrics to the monitoring server.
func sendMetrics(metrics Metrics, serverURL string) error {
	jsonData, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %v", err)
	}

	resp, err := http.Post(serverURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send metrics: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("Metrics sent: %s\n", resp.Status)
	return nil
}

// collectMetrics gathers system metrics using gopsutil.
func collectMetrics() (Metrics, error) {
	hostname, err := getHostname()
	if err != nil {
		return Metrics{}, fmt.Errorf("failed to get hostname: %v", err)
	}
	ip, err := getLocalIP()
	if err != nil {
		return Metrics{}, fmt.Errorf("failed to get local IP: %v", err)
	}

	// Get CPU usage (averaged over one second)
	cpuPercents, err := cpu.Percent(time.Second, false)
	if err != nil || len(cpuPercents) == 0 {
		return Metrics{}, fmt.Errorf("failed to get CPU usage: %v", err)
	}
	cpuUsage := cpuPercents[0]

	// Get memory usage
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return Metrics{}, fmt.Errorf("failed to get memory usage: %v", err)
	}
	ramUsage := vmStat.UsedPercent

	// Get disk usage (for "/" mount point)
	diskStat, err := disk.Usage("/")
	if err != nil {
		return Metrics{}, fmt.Errorf("failed to get disk usage: %v", err)
	}
	diskUsage := diskStat.UsedPercent

	return Metrics{
		Hostname:  hostname,
		IP:        ip,
		Timestamp: time.Now().UnixMilli(),
		CPUUsage:  cpuUsage,
		DiskUsage: diskUsage,
		RAMUsage:  ramUsage,
	}, nil
}

func main() {
	// === Part 1: Agent Registration ===
	// Open a listener on a random port; ":0" assigns an available port.
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		fmt.Println("Error starting listener:", err)
		return
	}
	agentPort := ln.Addr().(*net.TCPAddr).Port

	// Start a dummy TCP server to keep the port open.
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				fmt.Println("Error accepting connection:", err)
				continue
			}
			// Optionally, handle the connection (e.g., log, read, respond)
			// For now, simply close it immediately.
			conn.Close()
		}
	}()

	hostname, err := getHostname()
	if err != nil {
		fmt.Println("Error getting hostname:", err)
		return
	}

	ip, err := getLocalIP()
	if err != nil {
		fmt.Println("Error getting local IP:", err)
		return
	}

	// Retrieve open ports based on the PORTS environment variable (or scan all if not set).
	openPorts := getOpenPorts()

	agentInfo := AgentInfo{
		Hostname:  hostname,
		IP:        ip,
		OpenPorts: openPorts,
		Timestamp: time.Now().UnixMilli(),
		AgentPort: agentPort,
	}

	// Build the server registration URL using environment variables.
	hostEnv := os.Getenv("MONITORING_SERVER_HOST")
	if hostEnv == "" {
		hostEnv = "localhost"
	}
	portEnv := os.Getenv("MONITORING_SERVER_PORT")
	if portEnv == "" {
		portEnv = "8080"
	}
	registrationURL := "http://" + hostEnv + ":" + portEnv + "/api/agent/register"
	fmt.Printf("Registering agent to: %s\n", registrationURL)

	if err := registerAgent(agentInfo, registrationURL); err != nil {
		fmt.Println("Error registering agent:", err)
		return
	}

	// === Part 2: Metrics Sending ===
	// Build the metrics endpoint URL.
	metricsURL := "http://" + hostEnv + ":" + portEnv + "/api/metrics"
	fmt.Printf("Sending metrics to: %s\n", metricsURL)

	// Read the send interval from the environment variable SEND_INTERVAL (in seconds).
	sendIntervalStr := os.Getenv("SEND_INTERVAL")
	sendInterval := 60 * time.Second // default value
	if sendIntervalStr != "" {
		if seconds, err := strconv.Atoi(sendIntervalStr); err == nil {
			sendInterval = time.Duration(seconds) * time.Second
		} else {
			fmt.Printf("Invalid SEND_INTERVAL value, using default 60 seconds: %v\n", err)
		}
	}

	ticker := time.NewTicker(sendInterval)
	defer ticker.Stop()

	// Send metrics immediately at startup.
	metrics, err := collectMetrics()
	if err != nil {
		fmt.Printf("Error collecting metrics: %v\n", err)
	} else {
		if err := sendMetrics(metrics, metricsURL); err != nil {
			fmt.Printf("Error sending metrics: %v\n", err)
		}
	}

	// Periodically send metrics.
	for range ticker.C {
		metrics, err := collectMetrics()
		if err != nil {
			fmt.Printf("Error collecting metrics: %v\n", err)
			continue
		}
		if err := sendMetrics(metrics, metricsURL); err != nil {
			fmt.Printf("Error sending metrics: %v\n", err)
		}
	}
}