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

## Architecture

### Three-Component System

1. **Server (`cmd/server`)** - Central orchestration service
   - Serves static UI from `ui/build/`
   - REST API at `/api/*`
   - SSE endpoint at `/api/events` for real-time updates
   - Coordinates test scheduling and execution

2. **Worker (`cmd/worker`)** - Lightweight test binary deployed to remote hosts
   - Three modes: `server` (TCP listener), `client` (throughput test), `ping` (latency test)
   - Deployed to `/tmp/hl-speedtest-worker` on target devices via SSH
   - Outputs JSON results to stdout

3. **Frontend (`ui/`)** - SvelteKit + Tailwind CSS dashboard
   - Static build served by Go server
   - Real-time updates via Server-Sent Events
   - Chart.js for history visualization

### Test Execution Flow

1. Scheduler triggers test (timer or manual `/api/test/ping/all`, `/api/test/speed/all`)
2. Orchestrator SSHs to source and target devices
3. Worker binary deployed if not present
4. Target runs worker in server mode, source runs client/ping mode
5. Results parsed from stdout JSON, saved to SQLite, broadcast via SSE

### Key Internal Packages

- `internal/orchestrator/` - SSH connections, worker deployment, test coordination
- `internal/db/` - SQLite operations, schema embedded via `//go:embed`
- `internal/api/` - HTTP handlers, SSE broadcasting
- `internal/config/` - Configuration structs (currently hardcoded defaults)

## Database

SQLite at `data/speedtest.db`. Schema auto-applied on startup from `internal/db/schema.sql`.

**Tables:** `devices`, `schedules`, `results`

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

## Testing Notes

- Test environment uses `docker-compose.test.yml` with mock SSH nodes
- SSH key generated in `test-env/keys/` by `make local-test`
- Server connects to nodes via hostnames `node1`, `node2` on internal Docker network
- The api_test.go references `NewRouter` which should be `NewHandler` (known issue)
