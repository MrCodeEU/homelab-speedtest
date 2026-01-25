package orchestrator

import (
	"testing"
)

func TestNewOrchestrator(t *testing.T) {
	orch := NewOrchestrator("/tmp/worker", 8090)
	if orch == nil {
		t.Fatal("NewOrchestrator returned nil")
	}
	if orch.WorkerBinaryPath != "/tmp/worker" {
		t.Errorf("Expected WorkerBinaryPath '/tmp/worker', got '%s'", orch.WorkerBinaryPath)
	}
	if orch.WorkerPort != 8090 {
		t.Errorf("Expected WorkerPort 8090, got %d", orch.WorkerPort)
	}
}

func TestNewOrchestratorDefaultPort(t *testing.T) {
	orch := NewOrchestrator("/tmp/worker", 0)
	if orch == nil {
		t.Fatal("NewOrchestrator returned nil")
	}
	if orch.WorkerPort != 8090 {
		t.Errorf("Expected default WorkerPort 8090 when 0 is passed, got %d", orch.WorkerPort)
	}
}

func TestWorkerResponseSerialization(t *testing.T) {
	resp := WorkerResponse{
		Success:       true,
		LatencyMs:     0.5,
		JitterMs:      0.1,
		PacketLoss:    0.0,
		BandwidthMbps: 1000.0,
	}

	if !resp.Success {
		t.Error("Expected Success to be true")
	}

	if resp.LatencyMs != 0.5 {
		t.Errorf("Expected LatencyMs 0.5, got %f", resp.LatencyMs)
	}
	if resp.JitterMs != 0.1 {
		t.Errorf("Expected JitterMs 0.1, got %f", resp.JitterMs)
	}
	if resp.PacketLoss != 0.0 {
		t.Errorf("Expected PacketLoss 0.0, got %f", resp.PacketLoss)
	}

	if resp.BandwidthMbps != 1000.0 {
		t.Errorf("Expected BandwidthMbps 1000.0, got %f", resp.BandwidthMbps)
	}
}
