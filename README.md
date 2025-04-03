# Cheetah Monitoring Agent Documentation

This document provides a detailed explanation of the Cheetah Monitoring Agent, a lightweight tool written in Go. The agent collects system metrics and registers itself with the monitoring server. It also supports customizable configuration of ports to report in the registration message.

---

## Overview

The agent performs the following operations:

### 1. Agent Registration

- **Listener:** At startup, the agent opens a TCP listener on a random port and starts a dummy TCP server that continuously accepts incoming connections. This ensures that the chosen port remains open and reachable.
- **Port Reporting:** The agent sends its registration data to the server at the `/api/agent/register` endpoint. The registration payload includes:
  - **Hostname**
  - **IP Address**
  - **Open Ports:**  
    If the environment variable `PORTS` is set, the agent uses exactly that list (which can include individual ports and ranges, e.g., `8080,22,27017` or `9000-9090`). If `PORTS` is not set, the agent scans all ports from 1 to 65535 and returns only those that are open.
  - **Timestamp**
  - **AgentPort:** The port on which the dummy TCP server is listening.
- **Status:** The status of the agent ("UP" or "DOWN") is managed by the server based on reachability checks.

### 2. Metrics Sending

- The agent collects system metrics (hostname, IP, timestamp, CPU usage, disk usage, and RAM usage) using the `gopsutil` library.
- These metrics are sent periodically to the monitoring server at the `/api/metrics` endpoint.
- The sending interval is configurable via the environment variable `SEND_INTERVAL`.

---

## Environment Variables

The agent behavior can be customized using the following environment variables:

- **PORTS:**  
  A comma-separated list of ports or port ranges to include in the registration message.  
  Examples:
  - `8080,22,27017`
  - `9000-9090,1433`

  If `PORTS` is set, the agent will send exactly those ports without verifying if they are open.  
  If not set, the agent will perform a full port scan (1-65535) and include only the ports that are open.

- **MONITORING_SERVER_HOST:**  
  The hostname or IP address of the monitoring server.  
  *Default:* `localhost`

- **MONITORING_SERVER_PORT:**  
  The port on which the monitoring server is running.  
  *Default:* `8080`

- **SEND_INTERVAL:**  
  The interval (in seconds) between sending metrics to the server.  
  *Default:* `60` seconds

---

## How It Works

### 1. Registration Phase

- **Listener Setup:**  
  The agent opens a TCP listener on a random port (using `:0`) and starts a dummy TCP server in a goroutine that accepts and immediately closes incoming connections. This ensures that the agent remains reachable on the chosen port (`agentPort`).

- **Data Collection for Registration:**  
  The agent gathers:
  - Hostname (via `os.Hostname()`)
  - Local IP address (via `net.InterfaceAddrs()`)
  - Open Ports:  
    - If `PORTS` is defined, it parses the provided string (supporting comma-separated lists and ranges) and returns that list.
    - Otherwise, it scans ports 1 to 65535 (using concurrent goroutines with a concurrency limit) and returns only the ports that are open.
  - Timestamp (current Unix time in milliseconds)
  - AgentPort (the port where the dummy server is listening)

- **Sending Registration:**  
  The collected data is marshaled into JSON and sent via an HTTP POST to the endpoint:  
  `http://<MONITORING_SERVER_HOST>:<MONITORING_SERVER_PORT>/api/agent/register`

### 2. Metrics Collection and Sending

- **Metrics Collection:**  
  The agent collects system metrics using the `gopsutil` library:
  - CPU usage (percentage, averaged over one second)
  - Memory usage (used percentage)
  - Disk usage (for the root mount point)

- **Sending Metrics:**  
  The collected metrics, along with hostname, IP, and timestamp, are sent periodically (based on `SEND_INTERVAL`) via an HTTP POST to:  
  `http://<MONITORING_SERVER_HOST>:<MONITORING_SERVER_PORT>/api/metrics`

---

## Running the Agent

1. **Clone the Repository:**

```bash
git clone https://github.com/edoardopelli/cheetah-monitoring-agent.git
cd cheetah-monitoring-agent
```

2. **Set Environment Variables:**

For example, in a Unix-like terminal:

```bash
export PORTS="8080,22,27017"
export MONITORING_SERVER_HOST="your.server.address"
export MONITORING_SERVER_PORT="8080"
export SEND_INTERVAL="60"
```

3. **Build and Run the Agent:**

Build the agent:

```bash
go build -o cheetah-monitoring-agent
```

Then run it:

```bash
./cheetah-monitoring-agent
```

The agent will start its dummy TCP server, register itself with the monitoring server, and begin sending system metrics periodically.

---

## Summary

- **Agent Registration:**  
  The agent registers itself with the monitoring server, sending hostname, IP, open ports (as defined by the PORTS variable or determined by scanning), timestamp, and the active listening port.

- **Metrics Sending:**  
  The agent periodically collects and sends system metrics to the monitoring server.

- **Configuration:**  
  The behavior is customizable via environment variables for ports, server address, and send interval.

- **Dummy Server:**  
  A dummy TCP server is kept active to ensure the agent’s port is open and reachable for health checks.

This documentation covers the main features and configuration of the Cheetah Monitoring Agent. For further details or customization, please refer to the source code or contact the project maintainer.
