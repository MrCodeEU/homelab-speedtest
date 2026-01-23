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

func (o *Orchestrator) RunSpeedTest(source, target db.Device) (*WorkerResponse, error) {
	log.Printf("[Orchestrator] Starting Speed Test: %s -> %s", source.Name, target.Name)

	sourceClient, err := ConnectSSH(source.SSHUser, source.Hostname, source.SSHPort, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to source %s: %w", source.Name, err)
	}
	defer func() { _ = sourceClient.Close() }()

	targetClient, err := ConnectSSH(target.SSHUser, target.Hostname, target.SSHPort, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to target %s: %w", target.Name, err)
	}
	defer func() { _ = targetClient.Close() }()

	if err = o.deployWorker(sourceClient); err != nil {
		return nil, fmt.Errorf("failed to deploy worker to source: %w", err)
	}
	if err = o.deployWorker(targetClient); err != nil {
		return nil, fmt.Errorf("failed to deploy worker to target: %w", err)
	}

	serverPort := 8090
	_, _, _ = targetClient.RunCommand(fmt.Sprintf("fuser -k %d/tcp || pkill -f 'mode server -port %d'", serverPort, serverPort))
	time.Sleep(1 * time.Second)

	serverCmd := fmt.Sprintf("/tmp/hl-speedtest-worker -mode server -port %d", serverPort)
	go func() {
		_, _, _ = targetClient.RunCommand(serverCmd)
	}()
	time.Sleep(3 * time.Second)

	targetAddr := target.Hostname
	if target.IP != "" {
		targetAddr = target.IP
	}

	clientCmd := fmt.Sprintf("/tmp/hl-speedtest-worker -mode client -target %s:%d", targetAddr, serverPort)
	stdout, stderr, errClient := sourceClient.RunCommand(clientCmd)

	// Cleanup
	_, _, _ = targetClient.RunCommand(fmt.Sprintf("fuser -k %d/tcp || pkill -f 'mode server -port %d'", serverPort, serverPort))

	if errClient != nil {
		return nil, fmt.Errorf("client failed: %w, stdout: %s, stderr: %s", errClient, stdout, stderr)
	}

	return o.parseWorkerOutput(stdout, stderr)
}

func (o *Orchestrator) RunPing(source, target db.Device) (*WorkerResponse, error) {
	log.Printf("[Orchestrator] Starting Ping Test: %s -> %s", source.Name, target.Name)

	sourceClient, err := ConnectSSH(source.SSHUser, source.Hostname, source.SSHPort, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to source %s: %w", source.Name, err)
	}
	defer func() { _ = sourceClient.Close() }()

	targetClient, err := ConnectSSH(target.SSHUser, target.Hostname, target.SSHPort, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to target %s: %w", target.Name, err)
	}
	defer func() { _ = targetClient.Close() }()

	if err = o.deployWorker(sourceClient); err != nil {
		return nil, fmt.Errorf("failed to deploy worker to source: %w", err)
	}
	if err = o.deployWorker(targetClient); err != nil {
		return nil, fmt.Errorf("failed to deploy worker to target: %w", err)
	}

	serverPort := 8090
	_, _, _ = targetClient.RunCommand(fmt.Sprintf("fuser -k %d/tcp || pkill -f 'mode server -port %d'", serverPort, serverPort))
	time.Sleep(1 * time.Second)

	serverCmd := fmt.Sprintf("/tmp/hl-speedtest-worker -mode server -port %d", serverPort)
	go func() {
		_, _, _ = targetClient.RunCommand(serverCmd)
	}()
	time.Sleep(3 * time.Second)

	targetAddr := target.Hostname
	if target.IP != "" {
		targetAddr = target.IP
	}

	cmd := fmt.Sprintf("/tmp/hl-speedtest-worker -mode ping -target %s:%d", targetAddr, serverPort)
	stdout, stderr, errPing := sourceClient.RunCommand(cmd)

	// Cleanup
	_, _, _ = targetClient.RunCommand(fmt.Sprintf("fuser -k %d/tcp || pkill -f 'mode server -port %d'", serverPort, serverPort))

	if errPing != nil {
		return nil, fmt.Errorf("ping failed: %w, stdout: %s, stderr: %s", errPing, stdout, stderr)
	}

	return o.parseWorkerOutput(stdout, stderr)
}

func (o *Orchestrator) deployWorker(client *SSHClient) error {
	remotePath := "/tmp/hl-speedtest-worker"
	if client.FileExists(remotePath) {
		return nil
	}
	return client.CopyFile(o.WorkerBinaryPath, remotePath, 0755)
}

func (o *Orchestrator) parseWorkerOutput(stdout, stderr string) (*WorkerResponse, error) {
	var resp WorkerResponse

	// We expect JSON on stdout. If it's empty but stderr has content, use that for error.
	if stdout == "" {
		return nil, fmt.Errorf("no output on stdout. stderr: %s", stderr)
	}

	jsonStart := -1
	for i := 0; i < len(stdout); i++ {
		if stdout[i] == '{' {
			jsonStart = i
			break
		}
	}

	if jsonStart == -1 {
		return nil, fmt.Errorf("no JSON found in stdout: %s (stderr: %s)", stdout, stderr)
	}

	if err := json.Unmarshal([]byte(stdout[jsonStart:]), &resp); err != nil {
		return nil, fmt.Errorf("json parse failed: %w, raw: %s (stderr: %s)", err, stdout, stderr)
	}

	if !resp.Success {
		return &resp, fmt.Errorf("worker reported failure: %s (stderr: %s)", resp.Error, stderr)
	}

	return &resp, nil
}
