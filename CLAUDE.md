# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Homelab Speedtest is a network performance monitoring tool for homelab environments. It measures bandwidth and latency between devices using a custom Go-based worker deployed via SSH, presenting results through a web interface.

## Build & Development Commands

### Go Backend
```bash
# Build worker binary (required before running server)
go build -o worker ./cmd/worker

# Run server (requires worker binary and ui/build to exist)
go run ./cmd/server

# Run all Go tests
go test ./...

# Run tests for a specific package
go test ./internal/db/...

# Format Go code
gofmt -w .
```

### Frontend (SvelteKit)
```bash
cd ui
npm install
npm run dev          # Development server
npm run build        # Production build to ui/build/
npm run check        # Type checking with svelte-check
```

### Docker (Full Stack)
```bash
# Build and run production image
docker build -t homelab-speedtest .
docker run -p 8080:8080 homelab-speedtest

# Local test environment with mock SSH nodes
make local-test      # Sets up server + 2 test nodes with SSH
make clean-test      # Tears down test environment
```

## Configuration

Configuration is done via environment variables in `cmd/server/main.go`:

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | `8080` | HTTP server port |
| `DATABASE_PATH` | `data/speedtest.db` | SQLite database file path |
| `WORKER_PORT` | `8090` | Port used by worker binary for bandwidth/ping tests |
| `PING_SCHEDULE` | `1m` | Default ping schedule (Go duration format) |
| `SPEEDTEST_SCHEDULE` | `15m` | Default speed test schedule (Go duration format) |

**Note:** Schedules are auto-seeded on first startup if none exist in the database.

## Architecture

### Three-Component System

1. **Server (`cmd/server`)** - Central orchestration service
   - Serves static UI from `ui/build/`
   - REST API at `/api/*`
   - SSE endpoint at `/api/events` for real-time updates
   - Coordinates test scheduling and execution
   - Reads configuration from environment variables

2. **Worker (`cmd/worker`)** - Lightweight test binary deployed to remote hosts
   - Three modes: `server` (TCP listener), `client` (throughput test), `ping` (latency test)
   - Deployed to `/tmp/hl-speedtest-worker` on target devices via SSH
   - Outputs JSON results to stdout
   - Listens on configurable port (default 8090, set via `WORKER_PORT`)

3. **Frontend (`ui/`)** - SvelteKit + Tailwind CSS dashboard
   - Static build served by Go server
   - Real-time updates via Server-Sent Events
   - Chart.js for history visualization

### Test Execution Flow

1. Scheduler triggers test (timer or manual `/api/test/ping/all`, `/api/test/speed/all`)
2. Orchestrator SSHs to source and target devices
3. Worker binary deployed if not present
4. Target runs worker in server mode on `WORKER_PORT`, source runs client/ping mode
5. Results parsed from stdout JSON, saved to SQLite, broadcast via SSE

### Key Internal Packages

- `internal/orchestrator/` - SSH connections, worker deployment, test coordination
  - `Orchestrator` struct holds `WorkerPort` configuration
  - `NewOrchestrator(workerPath string, workerPort int)` constructor
- `internal/db/` - SQLite operations, schema embedded via `//go:embed`
- `internal/api/` - HTTP handlers, SSE broadcasting
- `internal/config/` - Configuration structs

## Database

SQLite at `data/speedtest.db` (configurable via `DATABASE_PATH`). Schema auto-applied on startup from `internal/db/schema.sql`.

**Tables:** `devices`, `schedules`, `results`, `notification_settings`, `alert_rules`

Schedule `cron` field uses Go duration format (e.g., `1m`, `15m`) not actual cron syntax.

## API Endpoints

- `GET/POST /api/devices` - Device CRUD
- `DELETE /api/devices/{id}`
- `GET/PUT /api/schedules` - Schedule config
- `GET /api/results/latest` - Latest result per device pair
- `GET /api/history?limit=N` - Historical results
- `POST /api/test/ping/all` - Trigger all ping tests
- `POST /api/test/speed/all` - Trigger all speed tests
- `GET /api/events` - SSE stream for real-time updates
- `GET/POST/PUT/DELETE /api/alerts` - Alert rule management
- `GET/PUT /api/notifications/settings` - Notification settings (ntfy)

## Common Issues

### "no route to host" / Connection Errors

The worker port (default 8090) must be open on target devices. Error messages now include firewall hints:
- iptables: `sudo iptables -A INPUT -p tcp --dport 8090 -j ACCEPT`
- firewalld: `sudo firewall-cmd --add-port=8090/tcp --permanent && sudo firewall-cmd --reload`
- ufw: `sudo ufw allow 8090/tcp`

### Schedules Not Appearing in UI

Schedules are auto-seeded on startup. If missing, restart server or use API:
```bash
curl -X PUT http://localhost:8080/api/schedules \
  -H "Content-Type: application/json" \
  -d '{"type":"ping","cron":"1m","enabled":true}'
```

## Testing Notes

- Test environment uses `docker-compose.test.yml` with mock SSH nodes
- SSH key generated in `test-env/keys/` by `make local-test`
- Server connects to nodes via hostnames `node1`, `node2` on internal Docker network
- `NewOrchestrator` requires two arguments: `(workerPath string, workerPort int)`
