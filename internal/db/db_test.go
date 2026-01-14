package db

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/user/homelab-speedtest/internal/config"
)

func TestNew(t *testing.T) {
	// Create temp dir for test db
	tmpDir, err := os.MkdirTemp("", "speedtest-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	dbPath := filepath.Join(tmpDir, "test.db")
	cfg := config.DatabaseConfig{Path: dbPath}

	db, errNew := New(cfg)
	if errNew != nil {
		t.Fatalf("Failed to create database: %v", errNew)
	}
	defer func() { _ = db.Close() }()

	// Verify we can query
	if _, err = db.GetDevices(); err != nil {
		t.Fatalf("Failed to get devices: %v", err)
	}
}

func TestAddAndGetDevice(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "speedtest-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	dbPath := filepath.Join(tmpDir, "test.db")
	cfg := config.DatabaseConfig{Path: dbPath}

	db, errNew := New(cfg)
	if errNew != nil {
		t.Fatalf("Failed to create database: %v", errNew)
	}
	defer func() { _ = db.Close() }()

	// Add a device
	dev := Device{
		Name:     "TestNAS",
		Hostname: "nas.local",
		IP:       "100.64.0.1",
		SSHUser:  "root",
		SSHPort:  22,
	}

	if err = db.AddDevice(dev); err != nil {
		t.Fatalf("Failed to add device: %v", err)
	}

	// Get devices
	devices, errGet := db.GetDevices()
	if errGet != nil {
		t.Fatalf("Failed to get devices: %v", errGet)
	}

	if len(devices) != 1 {
		t.Fatalf("Expected 1 device, got %d", len(devices))
	}

	if devices[0].Name != "TestNAS" {
		t.Errorf("Expected name 'TestNAS', got '%s'", devices[0].Name)
	}

	if devices[0].Hostname != "nas.local" {
		t.Errorf("Expected hostname 'nas.local', got '%s'", devices[0].Hostname)
	}
}

func TestAddResult(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "speedtest-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	dbPath := filepath.Join(tmpDir, "test.db")
	cfg := config.DatabaseConfig{Path: dbPath}

	db, errNew := New(cfg)
	if errNew != nil {
		t.Fatalf("Failed to create database: %v", errNew)
	}
	defer func() { _ = db.Close() }()

	// Add two devices first
	_ = db.AddDevice(Device{Name: "Source", Hostname: "src.local", SSHUser: "root", SSHPort: 22})
	_ = db.AddDevice(Device{Name: "Target", Hostname: "dst.local", SSHUser: "root", SSHPort: 22})

	// Add a result
	errResult := db.AddResult(1, 2, "ping", 0.5, 0.1, 0.0, 0)
	if errResult != nil {
		t.Fatalf("Failed to add result: %v", errResult)
	}

	// Add speed result
	errSpeed := db.AddResult(1, 2, "speed", 0, 0, 0, 950.5)
	if errSpeed != nil {
		t.Fatalf("Failed to add speed result: %v", errSpeed)
	}
}
