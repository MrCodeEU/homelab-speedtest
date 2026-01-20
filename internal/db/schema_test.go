package db

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/user/homelab-speedtest/internal/config"
)

func TestSchemaIdempotency(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "speedtest-schema-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	dbPath := filepath.Join(tmpDir, "test.db")
	cfg := config.DatabaseConfig{Path: dbPath}

	// First initialization
	db1, err := New(cfg)
	if err != nil {
		t.Fatalf("First initialization failed: %v", err)
	}
	_ = db1.Close()

	// Second initialization (should not fail)
	db2, err := New(cfg)
	if err != nil {
		t.Fatalf("Second initialization failed (schema likely not idempotent): %v", err)
	}
	_ = db2.Close()
}
