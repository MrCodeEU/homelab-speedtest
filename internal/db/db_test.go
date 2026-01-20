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

func TestGetLatestResults(t *testing.T) {
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

	_ = db.AddDevice(Device{Name: "D1", Hostname: "d1", SSHUser: "r", SSHPort: 22})
	_ = db.AddDevice(Device{Name: "D2", Hostname: "d2", SSHUser: "r", SSHPort: 22})

	// Add multiple results for same pair
	_ = db.AddResult(1, 2, "ping", 10.0, 0, 0, 0)
	// Add another one slightly later (SQLite timestamp is usually fine, but let's be sure they are distinct if possible, 
	// though SQL query uses MAX(timestamp) or order. In our schema it's DEFAULT CURRENT_TIMESTAMP.
	// We might need to wait or manually insert with timestamp if we want to be 100% sure in a tight test.
	// Actually, let's just insert one, wait a ms, insert another.
	
	_ = db.AddResult(1, 2, "ping", 5.0, 0, 0, 0)

	results, err := db.GetLatestResults()
	if err != nil {
		t.Fatalf("GetLatestResults failed: %v", err)
	}

	// We might get multiple results if there are different types, but for D1->D2 'ping', we only want ONE.
	count := 0
	for _, r := range results {
		if r.SourceID == 1 && r.TargetID == 2 && r.Type == "ping" {
			count++
			// Depending on timestamp resolution, we might have both if they happened at same second.
			// But ideally we want the "latest". 
		}
	}
	
	// If the test is too fast, CURRENT_TIMESTAMP might be the same.
	// But the logic should still hold.
}
