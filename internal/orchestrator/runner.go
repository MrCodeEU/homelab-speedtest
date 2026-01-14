package orchestrator

import (
	"encoding/json"
	"fmt"
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
	// 1. Connect to Source
	sourceClient, err := ConnectSSH(source.SSHUser, source.Hostname, source.SSHPort, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to source %s: %w", source.Name, err)
	}
	defer sourceClient.Close()

	// 2. Connect to Target
	targetClient, err := ConnectSSH(target.SSHUser, target.Hostname, target.SSHPort, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to target %s: %w", target.Name, err)
	}
	defer targetClient.Close()

	// 3. Deploy Worker
	if err := o.deployWorker(sourceClient); err != nil {
		return nil, fmt.Errorf("failed to deploy worker to source: %w", err)
	}
	if err := o.deployWorker(targetClient); err != nil {
		return nil, fmt.Errorf("failed to deploy worker to target: %w", err)
	}

	// 4. Start Server on Target
	serverPort := 8090
	serverCmd := fmt.Sprintf("/tmp/hl-speedtest-worker -mode server -port %d", serverPort)

	// Start server in background. We use a goroutine to start it,
	// but we must rely on the command not blocking forever or using a mechanism to background it on the remote shell.
	// "nohup ... &" is safer.
	go func() {
		// This might block until cancelled or connection closed
		targetClient.RunCommand(serverCmd)
	}()
	time.Sleep(2 * time.Second) // Wait for server start

	// 5. Start Client on Source
	// We need target's address reachable from source.
	targetAddr := target.Hostname // Assuming this is reachable (Tailscale IP/Host)
	if target.IP != "" {
		targetAddr = target.IP
	}

	clientCmd := fmt.Sprintf("/tmp/hl-speedtest-worker -mode client -target %s:%d", targetAddr, serverPort)
	output, err := sourceClient.RunCommand(clientCmd)

	// 6. Cleanup (Kill server on target)
	// We do this regardless of client success
	go targetClient.RunCommand("pkill -f hl-speedtest-worker")

	if err != nil {
		return nil, fmt.Errorf("client failed: %w, output: %s", err, output)
	}

	// 7. Parse Result
	var resp WorkerResponse
	if err := json.Unmarshal([]byte(output), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse output: %w, raw: %s", err, output)
	}

	return &resp, nil
}

// RunPing coordinates a ping test from source to target.
func (o *Orchestrator) RunPing(source, target db.Device) (*WorkerResponse, error) {
	client, err := ConnectSSH(source.SSHUser, source.Hostname, source.SSHPort, nil)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	if err := o.deployWorker(client); err != nil {
		return nil, err
	}

	targetAddr := target.Hostname
	if target.IP != "" {
		targetAddr = target.IP
	}

	cmd := fmt.Sprintf("/tmp/hl-speedtest-worker -mode ping -target %s", targetAddr)
	output, err := client.RunCommand(cmd)
	if err != nil {
		return nil, err
	}

	var resp WorkerResponse
	if err := json.Unmarshal([]byte(output), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (o *Orchestrator) deployWorker(client *SSHClient) error {
	remotePath := "/tmp/hl-speedtest-worker"
	return client.CopyFile(o.WorkerBinaryPath, remotePath, 0755)
}
