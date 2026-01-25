package orchestrator

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/user/homelab-speedtest/internal/db"
)

type Orchestrator struct {
	WorkerBinaryPath string
	WorkerPort       int
}

func NewOrchestrator(workerPath string, workerPort int) *Orchestrator {
	if workerPort <= 0 {
		workerPort = 8090 // default port
	}
	return &Orchestrator{
		WorkerBinaryPath: workerPath,
		WorkerPort:       workerPort,
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

	_, _, _ = targetClient.RunCommand(fmt.Sprintf("fuser -k %d/tcp || pkill -f 'mode server -port %d'", o.WorkerPort, o.WorkerPort))
	time.Sleep(1 * time.Second)

	serverCmd := fmt.Sprintf("/tmp/hl-speedtest-worker -mode server -port %d", o.WorkerPort)
	go func() {
		_, _, _ = targetClient.RunCommand(serverCmd)
	}()
	time.Sleep(3 * time.Second)

	targetAddr := target.Hostname
	if target.IP != "" {
		targetAddr = target.IP
	}

	clientCmd := fmt.Sprintf("/tmp/hl-speedtest-worker -mode client -target %s:%d", targetAddr, o.WorkerPort)
	stdout, stderr, errClient := sourceClient.RunCommand(clientCmd)

	// Cleanup
	_, _, _ = targetClient.RunCommand(fmt.Sprintf("fuser -k %d/tcp || pkill -f 'mode server -port %d'", o.WorkerPort, o.WorkerPort))

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

	_, _, _ = targetClient.RunCommand(fmt.Sprintf("fuser -k %d/tcp || pkill -f 'mode server -port %d'", o.WorkerPort, o.WorkerPort))
	time.Sleep(1 * time.Second)

	serverCmd := fmt.Sprintf("/tmp/hl-speedtest-worker -mode server -port %d", o.WorkerPort)
	go func() {
		_, _, _ = targetClient.RunCommand(serverCmd)
	}()
	time.Sleep(3 * time.Second)

	targetAddr := target.Hostname
	if target.IP != "" {
		targetAddr = target.IP
	}

	cmd := fmt.Sprintf("/tmp/hl-speedtest-worker -mode ping -target %s:%d", targetAddr, o.WorkerPort)
	stdout, stderr, errPing := sourceClient.RunCommand(cmd)

	// Cleanup
	_, _, _ = targetClient.RunCommand(fmt.Sprintf("fuser -k %d/tcp || pkill -f 'mode server -port %d'", o.WorkerPort, o.WorkerPort))

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
		errMsg := o.enhanceErrorMessage(resp.Error, stderr)
		return &resp, fmt.Errorf("worker reported failure: %s", errMsg)
	}

	return &resp, nil
}

// enhanceErrorMessage adds helpful hints for common connection errors
func (o *Orchestrator) enhanceErrorMessage(errMsg, stderr string) string {
	combined := errMsg + " " + stderr

	// Check for common connection errors and add hints
	if strings.Contains(combined, "no route to host") {
		return fmt.Sprintf("%s (stderr: %s) [Hint: Check if the target device's firewall allows incoming connections on port %d. "+
			"For iptables: 'sudo iptables -A INPUT -p tcp --dport %d -j ACCEPT'. "+
			"For firewalld: 'sudo firewall-cmd --add-port=%d/tcp --permanent && sudo firewall-cmd --reload'. "+
			"For ufw: 'sudo ufw allow %d/tcp']",
			errMsg, stderr, o.WorkerPort, o.WorkerPort, o.WorkerPort, o.WorkerPort)
	}

	if strings.Contains(combined, "connection refused") {
		return fmt.Sprintf("%s (stderr: %s) [Hint: The worker server may not be running on port %d. "+
			"This could indicate: 1) The worker failed to start on the target device, "+
			"2) The target IP/hostname is incorrect, or "+
			"3) A firewall is blocking the connection]",
			errMsg, stderr, o.WorkerPort)
	}

	if strings.Contains(combined, "connection timed out") || strings.Contains(combined, "i/o timeout") {
		return fmt.Sprintf("%s (stderr: %s) [Hint: Connection timed out to port %d. "+
			"Check network connectivity between devices and ensure firewall rules allow traffic on this port]",
			errMsg, stderr, o.WorkerPort)
	}

	// Default: just return with stderr
	return fmt.Sprintf("%s (stderr: %s)", errMsg, stderr)
}
