# Homelab Speedtest Project Context

## Project Overview

**Homelab Speedtest** is a centralized tool designed to monitor network performance within a homelab environment. It measures bandwidth and latency between devices using a custom Go-based worker and presents the results via a web interface.

### Architecture

The system consists of three main components:

1.  **Server (`cmd/server`)**: The central control unit.
    *   **Role**: Serves the web UI, manages the SQLite database, exposes a REST API, and orchestrates tests.
    *   **Tech**: Go 1.25.5 (Standard Library + `modernc.org/sqlite`, `gorilla/websocket`).
    *   **Database**: SQLite (`data/speedtest.db`). Stores devices, schedules, test results, notification settings, and alert rules.

2.  **Worker (`cmd/worker`)**: A lightweight binary deployed to target devices.
    *   **Role**: Performs the actual network tests.
    *   **Modes**:
        *   `server`: Listens on a TCP port to receive traffic.
        *   `client`: Connects to a target worker (in server mode) to measure throughput.
        *   `ping`: Measures latency to a target.
    *   **Deployment**: The server uses SSH to copy/execute this binary on remote hosts.

3.  **Frontend (`ui/`)**: The user interface.
    *   **Role**: Dashboard for viewing results and managing configuration.
    *   **Tech**: SvelteKit 2, Svelte 5, Tailwind CSS 4, Vite 7, Chart.js 4.
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
    *   `worker/`: Main worker application.
*   `internal/`: Private application code.
    *   `api/`: HTTP API handlers (REST + WebSocket + SSE).
    *   `config/`: Configuration loading.
    *   `db/`: Database access and schema (`schema.sql`).
    *   `orchestrator/`: Logic for scheduling and running tests (SSH, Protocol).
    *   `notify/`: Notification logic (Email + ntfy).
*   `ui/`: SvelteKit frontend source code.
*   `Dockerfile`: Multi-stage build definition.

## Key Conventions

*   **Database**: SQLite (`modernc.org/sqlite`). Schema defined in `internal/db/schema.sql`.
*   **Orchestration**: The system relies on SSH to manage remote workers.
*   **API**: RESTful API at `/api`. Real-time updates via SSE (`/api/events`) and WebSocket (`/api/ws`).
*   **Frontend**: Svelte 5 with Tailwind CSS v4.

## Current State & TODOs
*   **Notifications**: Support for Email (SMTP) and ntfy is implemented in the backend (`internal/notify`) and API (`/notification-settings`). UI exists but needs test buttons.
*   **Alerts**: Configurable rules for speed, ping, packet loss, and errors. Backend fully supports CRUD (`/alert-rules`). UI allows adding/deleting but needs editing.
*   **Devices/Alerts Editing**: Backend supports `PUT /devices/{id}` and `PUT /alert-rules/{id}`. UI currently only supports Delete/Re-add.
*   **Speed Diagram**: Users report it only shows the last test, not history. Needs investigation (likely UI chart logic or data retrieval limit).
*   **Test Notifications**: Missing "Test" button for notification settings in the UI, though endpoints `POST /notify/test/ntfy` and `POST /notify/test/email` exist.
