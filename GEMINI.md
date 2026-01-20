# Homelab Speedtest Project Context

## Project Overview

**Homelab Speedtest** is a centralized tool designed to monitor network performance within a homelab environment. It measures bandwidth and latency between devices using a custom Go-based worker and presents the results via a web interface.

### Architecture

The system consists of three main components:

1.  **Server (`cmd/server`)**: The central control unit.
    *   **Role**: Serves the web UI, manages the SQLite database, exposes a REST API, and orchestrates tests.
    *   **Tech**: Go (Standard Library + `modernc.org/sqlite`).
    *   **Database**: SQLite (`data/speedtest.db`). Stores devices, schedules, and test results.

2.  **Worker (`cmd/worker`)**: A lightweight binary deployed to target devices.
    *   **Role**: Performs the actual network tests.
    *   **Modes**:
        *   `server`: Listens on a TCP port to receive traffic.
        *   `client`: Connects to a target worker (in server mode) to measure throughput.
        *   `ping`: Measures latency to a target (currently in development).
    *   **Deployment**: The server likely uses SSH to copy/execute this binary on remote hosts.

3.  **Frontend (`ui/`)**: The user interface.
    *   **Role**: Dashboard for viewing results and managing configuration.
    *   **Tech**: SvelteKit, Tailwind CSS, Vite.
    *   **Serving**: Built as static assets (`ui/build`) and served by the Go server's `http.FileServer`.

## Building and Running

### Prerequisites
*   **Go**: 1.25+
*   **Node.js**: 20+ (for UI)
*   **Docker**: Optional, for full system build.

### Development

#### 1. Backend (Server + Worker)
You need to build the worker first so the server can use it (or configure the server to look for it).

```bash
# Build the worker binary
go build -o worker ./cmd/worker

# Run the server (ensure 'ui/build' exists or update code to skip if just testing API)
# The server expects the 'worker' binary in the working directory by default.
go run ./cmd/server
```

#### 2. Frontend (UI)
Run the SvelteKit development server with proxy config (if applicable) or pointing to the Go API.

```bash
cd ui
npm install
npm run dev
```

To build the UI for the Go server to serve:
```bash
cd ui
npm run build
```

### Full Production Build (Docker)
The `Dockerfile` handles the multi-stage build process:
1.  Builds the Svelte UI.
2.  Builds the Worker binary.
3.  Builds the Server binary.
4.  Assembles a final Alpine image with all components.

```bash
docker build -t homelab-speedtest .
docker run -p 8080:8080 homelab-speedtest
```

## Project Structure

*   `cmd/`: Entry points for the applications.
    *   `server/`: Main server application.
    *   `worker/`: Network testing utility.
*   `internal/`: Private application code.
    *   `api/`: HTTP API handlers.
    *   `config/`: Configuration loading.
    *   `db/`: Database access and schema (`schema.sql`).
    *   `orchestrator/`: Logic for scheduling and running tests (SSH, Protocol).
    *   `notify/`: (Likely) Notification logic.
*   `ui/`: SvelteKit frontend source code.
*   `Dockerfile`: Multi-stage build definition.

## Key Conventions

*   **Database**: SQLite is used for simplicity and portability. Migrations are likely manual or handled on startup (schema defined in `internal/db/schema.sql`).
*   **Orchestration**: The system relies on SSH (implied by `ssh.go` and `openssh-client` in Docker) to manage remote workers.
*   **API**: RESTful API at `/api`.
*   **Frontend**: Svelte 5 (implied by recent versions) with Tailwind CSS v4.

## Future Context / TODOs
*   **Ping Implementation**: The `worker`'s ping mode is currently a placeholder and needs robust implementation (ICMP or TCP connect).
*   **SSH Integration**: Verify how the orchestrator uses SSH to deploy the worker.
*   **UI Integration**: Ensure the UI properly communicates with the `/api` endpoints.
