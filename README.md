# Cheetah Monitoring Agent

This project is a lightweight monitoring agent written in Go. It collects system metrics (hostname, IP, timestamp, CPU usage, disk usage, and RAM usage) and sends them to a monitoring server via an HTTP POST request.

## Features

- **Metrics Collection:** Uses `gopsutil` to gather system metrics.
- **Configurable:** Server URL and data sending interval are set via environment variables.
- **JSON Format:** Metrics are sent in JSON format.

## Prerequisites

- [Go](https://golang.org/dl/) (version 1.16 or later recommended)
- A running monitoring server (e.g., your Spring Boot server)
- (Optional) MongoDB if testing with the provided server example

## Installation and Running

1. **Clone the repository:**

   ```bash
   git clone https://github.com/your-username/cheetah-monitoring-agent.git
   cd cheetah-monitoring-agent
   ```

2. **Set Environment Variables:**
   - `MONITORING_SERVER_URL`: The URL of the monitoring server endpoint (e.g., http://localhost:8080/api/metrics)
   - `SEND_INTERVAL`: The interval in seconds between sending metrics (default is 60 seconds)

   For example, on Linux/Mac:

   ```bash
   export MONITORING_SERVER_URL="http://localhost:8080/api/metrics"
   export SEND_INTERVAL="60"
   ```

   On Windows (Command Prompt):

   ```cmd
   set MONITORING_SERVER_URL=http://localhost:8080/api/metrics
   set SEND_INTERVAL=60
   ```

3. **Run the Agent:**

   In the project directory, run:

   ```bash
   go run main.go
   ```

   The agent will start collecting system metrics and send them to the specified monitoring server at the defined interval.

4. **Build the Agent (Optional):**

   To build the agent as a standalone binary:

   ```bash
   go build -o cheetah-monitoring-agent
   ```

   Then run the binary:

   ```bash
   ./cheetah-monitoring-agent
   ```

## License

This project is licensed under the MIT License.
