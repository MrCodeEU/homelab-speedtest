package orchestrator

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/user/homelab-speedtest/internal/db"
)

type Orchestrator struct {
	WorkerBinaryPath string
}

func NewOrchestrator(workerPath string) *Orchestrator {
	return &Orchestrator{
		WorkerBinaryPath: workerPath,
	}
}

// RunSpeedTest coordinates a speed test from source to target.
func (o *Orchestrator) RunSpeedTest(source, target db.Device) (*WorkerResponse, error) {
	log.Printf("[Orchestrator] Starting Speed Test: %s -> %s", source.Name, target.Name)

	// 1. Connect to Source
	log.Printf("[Orchestrator] Connecting to source %s (%s:%d)...", source.Name, source.Hostname, source.SSHPort)
	sourceClient, err := ConnectSSH(source.SSHUser, source.Hostname, source.SSHPort, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to source %s: %w", source.Name, err)
	}
	defer func() { _ = sourceClient.Close() }()

	// 2. Connect to Target
	log.Printf("[Orchestrator] Connecting to target %s (%s:%d)...", target.Name, target.Hostname, target.SSHPort)
	targetClient, err := ConnectSSH(target.SSHUser, target.Hostname, target.SSHPort, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to target %s: %w", target.Name, err)
	}
	defer func() { _ = targetClient.Close() }()

	// 3. Deploy Worker
	log.Printf("[Orchestrator] Deploying worker to source %s...", source.Name)
	if err = o.deployWorker(sourceClient); err != nil {
		return nil, fmt.Errorf("failed to deploy worker to source: %w", err)
	}
	log.Printf("[Orchestrator] Deploying worker to target %s...", target.Name)
	if err = o.deployWorker(targetClient); err != nil {
		return nil, fmt.Errorf("failed to deploy worker to target: %w", err)
	}

	// 4. Start Server on Target
	serverPort := 8090
	serverCmd := fmt.Sprintf("/tmp/hl-speedtest-worker -mode server -port %d", serverPort)
	log.Printf("[Orchestrator] Starting worker server on target %s: %s", target.Name, serverCmd)

	// Start server in background.
	go func() {
		_, _ = targetClient.RunCommand(serverCmd)
	}()
	time.Sleep(2 * time.Second) // Wait for server start

	// 5. Start Client on Source
	targetAddr := target.Hostname // Assuming this is reachable (Tailscale IP/Host)
	if target.IP != "" {
		targetAddr = target.IP
	}

	clientCmd := fmt.Sprintf("/tmp/hl-speedtest-worker -mode client -target %s:%d", targetAddr, serverPort)
	log.Printf("[Orchestrator] Running worker client on source %s: %s", source.Name, clientCmd)
	output, errClient := sourceClient.RunCommand(clientCmd)

	// 6. Cleanup (Kill server on target)
	go func() { _, _ = targetClient.RunCommand("pkill -f hl-speedtest-worker") }()

	if errClient != nil {
		log.Printf("[Orchestrator] Speed test client failed. Output: %s", output)
		return nil, fmt.Errorf("client failed: %w, output: %s", errClient, output)
	}

	log.Printf("[Orchestrator] Speed test raw output: %s", output)

	// 7. Parse Result
	var resp WorkerResponse
	if err = json.Unmarshal([]byte(output), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse output: %w, raw: %s", err, output)
	}

	log.Printf("[Orchestrator] Speed Test Completed: %s -> %s | Bandwidth: %.2f Mbps", source.Name, target.Name, resp.BandwidthMbps)
	return &resp, nil
}

// RunPing coordinates a ping test from source to target.
func (o *Orchestrator) RunPing(source, target db.Device) (*WorkerResponse, error) {
	log.Printf("[Orchestrator] Starting Ping Test: %s -> %s", source.Name, target.Name)

	log.Printf("[Orchestrator] Connecting to source %s...", source.Name)
	client, err := ConnectSSH(source.SSHUser, source.Hostname, source.SSHPort, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = client.Close() }()

	log.Printf("[Orchestrator] Deploying worker to source %s...", source.Name)
	if err = o.deployWorker(client); err != nil {
		return nil, err
	}

	targetAddr := target.Hostname
	if target.IP != "" {
		targetAddr = target.IP
	}

	// Target the worker server port (8090) for TCP ping
	cmd := fmt.Sprintf("/tmp/hl-speedtest-worker -mode ping -target %s:8090", targetAddr)
	log.Printf("[Orchestrator] Running ping command on source: %s", cmd)
	output, errPing := client.RunCommand(cmd)
	if errPing != nil {
		log.Printf("[Orchestrator] Ping command failed. Output: %s", output)
		return nil, errPing
	}

	log.Printf("[Orchestrator] Ping raw output: %s", output)

	var resp WorkerResponse
	if err = json.Unmarshal([]byte(output), &resp); err != nil {
		return nil, err
	}
	
	log.Printf("[Orchestrator] Ping Test Completed: %s -> %s | Latency: %.2f ms", source.Name, target.Name, resp.LatencyMs)
	return &resp, nil
}

func (o *Orchestrator) deployWorker(client *SSHClient) error {
	remotePath := "/tmp/hl-speedtest-worker"
	return client.CopyFile(o.WorkerBinaryPath, remotePath, 0755)
}
