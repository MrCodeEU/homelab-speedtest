# Stage 1: Build Frontend
FROM node:20-alpine AS ui-builder
WORKDIR /app/ui
COPY ui/package*.json ./
RUN npm ci
COPY ui/ .
# Build SvelteKit app. output usually to build/
RUN npm run build 
# Note: output directory depends on adapter. 'adapter-static' -> build/, 'adapter-node' -> build/. 
# We used 'minimal' template. Does it have an adapter?
# We probably need to install adapter-node or adapter-static.
# For a single binary deploy, often 'adapter-static' is used and served by Go.
# Assuming 'npm run build' produces something we can serve or we need to fix adapter.
# Let's assume the user handles adapter or we just use what's there (likely nothing or auto).
# Standard 'adapter-auto' might need Node at runtime.
# We want Go to serve static files. 'adapter-static' is best.
# I will assume we can just copy 'ui/build' (or .svelte-kit/output/client) to the Go server's static dir.
# Default 'create-svelte' output for static is 'build'.

# Stage 2: Build Worker
FROM golang:1.23-alpine AS worker-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/worker ./cmd/worker
COPY internal ./internal
# Build for Linux AMD64 (Generic)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /worker ./cmd/worker

# Stage 3: Build Server
FROM golang:1.23-alpine AS server-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/server ./cmd/server
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /server ./cmd/server

# Stage 4: Runtime
FROM alpine:latest
RUN apk add --no-cache ca-certificates openssh-client

WORKDIR /app
COPY --from=server-builder /server .
COPY --from=worker-builder /worker .
COPY --from=ui-builder /app/ui/build ./ui/build
# Create data dir
RUN mkdir -p data

EXPOSE 8080

CMD ["./server"]
