package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
)

// Metrics represents the system metrics to be sent.
type Metrics struct {
	Hostname  string  `json:"hostname"`
	IP        string  `json:"ip"`
	Timestamp int64   `json:"timestamp"`
	CPUUsage  float64 `json:"cpuUsage"`
	DiskUsage float64 `json:"diskUsage"`
	RAMUsage  float64 `json:"ramUsage"`
}

// collectMetrics gathers system metrics using gopsutil.
func collectMetrics() (Metrics, error) {
	// Get the hostname
	hostname, err := os.Hostname()
	if err != nil {
		return Metrics{}, fmt.Errorf("failed to get hostname: %v", err)
	}

	// Get local IP address
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
		Timestamp: time.Now().Unix(),
		CPUUsage:  cpuUsage,
		DiskUsage: diskUsage,
		RAMUsage:  ramUsage,
	}, nil
}

// getLocalIP returns the non-loopback local IP address.
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

// sendMetrics sends the collected metrics to the monitoring server via HTTP POST.
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

	fmt.Printf("Response status: %s\n", resp.Status)
	return nil
}

func main() {
	// Read server host and port from environment variables
	host := os.Getenv("MONITORING_SERVER_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("MONITORING_SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	// Build the full URL by concatenating host, port and the API endpoint path
	serverURL := "http://" + host + ":" + port + "/api/metrics"
	fmt.Printf("Using server URL: %s\n", serverURL)

	// Read send interval (in seconds) from environment variable SEND_INTERVAL
	sendIntervalStr := os.Getenv("SEND_INTERVAL")
	sendInterval := 60 * time.Second // default value
	if sendIntervalStr != "" {
		seconds, err := strconv.Atoi(sendIntervalStr)
		if err == nil {
			sendInterval = time.Duration(seconds) * time.Second
		} else {
			fmt.Printf("Invalid SEND_INTERVAL value, using default 60 seconds: %v\n", err)
		}
	}

	// Ticker for periodic metrics sending
	ticker := time.NewTicker(sendInterval)
	defer ticker.Stop()

	// Initial send on startup
	metrics, err := collectMetrics()
	if err != nil {
		fmt.Printf("Error collecting metrics: %v\n", err)
	} else {
		if err := sendMetrics(metrics, serverURL); err != nil {
			fmt.Printf("Error sending metrics: %v\n", err)
		}
	}

	// Loop to send metrics periodically
	for range ticker.C {
		metrics, err := collectMetrics()
		if err != nil {
			fmt.Printf("Error collecting metrics: %v\n", err)
			continue
		}
		if err := sendMetrics(metrics, serverURL); err != nil {
			fmt.Printf("Error sending metrics: %v\n", err)
		}
	}
}