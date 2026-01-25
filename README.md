# Homelab Speedtest

A network performance monitoring tool for homelab environments. Measures bandwidth and latency between devices using a custom Go-based worker deployed via SSH, presenting results through a web interface.

## Features

- **Bandwidth Testing**: Measures throughput between any two devices in your network
- **Latency Testing**: Ping tests with latency, jitter, and packet loss metrics
- **Automatic Scheduling**: Configure intervals for periodic testing
- **Real-time Updates**: Live dashboard with Server-Sent Events
- **Alert Rules**: Notifications via ntfy when thresholds are exceeded
- **Multi-device Support**: Test between any combination of SSH-accessible devices

## Quick Start

### Using Docker

```bash
docker build -t homelab-speedtest .
docker run -p 8080:8080 \
  -v ~/.ssh/id_rsa:/root/.ssh/id_rsa:ro \
  -v speedtest-data:/app/data \
  homelab-speedtest
```

### Manual Setup

```bash
# Build the worker binary
go build -o worker ./cmd/worker

# Build the frontend
cd ui && npm install && npm run build && cd ..

# Run the server
go run ./cmd/server
```

Access the web UI at `http://localhost:8080`

## Configuration

All configuration is done via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | `8080` | HTTP server port |
| `DATABASE_PATH` | `data/speedtest.db` | SQLite database file path |
| `WORKER_PORT` | `8090` | Port used by worker for tests |
| `PING_SCHEDULE` | `1m` | Default ping test interval (Go duration) |
| `SPEEDTEST_SCHEDULE` | `15m` | Default speed test interval (Go duration) |

### Example Docker Compose

```yaml
version: '3.8'
services:
  speedtest:
    build: .
    ports:
      - "8080:8080"
    environment:
      - WORKER_PORT=9000
      - PING_SCHEDULE=5m
      - SPEEDTEST_SCHEDULE=30m
    volumes:
      - ~/.ssh/id_rsa:/root/.ssh/id_rsa:ro
      - speedtest-data:/app/data

volumes:
  speedtest-data:
```

## Adding Devices

1. Navigate to the **Config** page in the web UI
2. Add devices with:
   - **Name**: Friendly identifier (e.g., "NAS", "NUC", "VPS")
   - **Hostname**: SSH-accessible hostname or IP
   - **IP** (optional): Override IP for test traffic (useful for Tailscale/WireGuard setups)
   - **SSH User**: Username for SSH connections
   - **SSH Port**: SSH port (default: 22)

The server must have SSH key-based access to all devices (no password prompts).

## Firewall Configuration

The worker uses a configurable TCP port (default: 8090) for tests. Each target device must allow incoming connections on this port.

```bash
# iptables
sudo iptables -A INPUT -p tcp --dport 8090 -j ACCEPT

# firewalld
sudo firewall-cmd --add-port=8090/tcp --permanent
sudo firewall-cmd --reload

# ufw
sudo ufw allow 8090/tcp
```

## Architecture

```
┌─────────────┐     SSH      ┌─────────────┐
│   Server    │─────────────▶│  Device A   │
│  (Go + UI)  │              │  (worker)   │
└─────────────┘              └──────┬──────┘
       │                            │
       │ SSH                        │ TCP :8090
       ▼                            ▼
┌─────────────┐              ┌─────────────┐
│  Device B   │◀─────────────│   Test      │
│  (worker)   │   bandwidth  │   Traffic   │
└─────────────┘    /ping     └─────────────┘
```

1. Server SSHs to both source and target devices
2. Deploys worker binary to `/tmp/hl-speedtest-worker` if not present
3. Starts worker in server mode on target device
4. Runs worker in client/ping mode on source device
5. Parses JSON results and stores in SQLite

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/devices` | List all devices |
| POST | `/api/devices` | Add a device |
| DELETE | `/api/devices/{id}` | Remove a device |
| GET | `/api/schedules` | Get schedule config |
| PUT | `/api/schedules` | Update a schedule |
| GET | `/api/results/latest` | Latest result per device pair |
| GET | `/api/history?limit=N` | Historical results |
| POST | `/api/test/ping/all` | Trigger all ping tests |
| POST | `/api/test/speed/all` | Trigger all speed tests |
| GET | `/api/events` | SSE stream for real-time updates |

## Troubleshooting

### "no route to host" errors

This typically means the target device's firewall is blocking the worker port. See [Firewall Configuration](#firewall-configuration).

### "connection refused" errors

- Worker may have failed to start on the target
- Check if the IP/hostname is correct
- Verify SSH connectivity: `ssh user@hostname`

### Tests not running

- Ensure the worker binary exists (`./worker`)
- Check that schedules are enabled in the Config page
- Verify SSH key authentication works without password prompts

### Schedule UI not showing

If schedules don't appear in the Config page, they may not have been seeded. The server now auto-creates default schedules on startup. Restart the server or manually trigger via API:

```bash
curl -X PUT http://localhost:8080/api/schedules \
  -H "Content-Type: application/json" \
  -d '{"type":"ping","cron":"1m","enabled":true}'
```

## Development

```bash
# Run tests
go test ./...

# Format code
gofmt -w .

# Frontend dev server (with hot reload)
cd ui && npm run dev

# Local test environment with mock SSH nodes
make local-test
make clean-test
```

## License

MIT
