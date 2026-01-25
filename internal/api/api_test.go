package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/user/homelab-speedtest/internal/config"
	"github.com/user/homelab-speedtest/internal/db"
	"github.com/user/homelab-speedtest/internal/notify"
	"github.com/user/homelab-speedtest/internal/orchestrator"
)

func TestGetLatestResultsAPI(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "api-test-*")
	defer func() { _ = os.RemoveAll(tmpDir) }()

	dbPath := filepath.Join(tmpDir, "test.db")
	database, _ := db.New(config.DatabaseConfig{Path: dbPath})
	defer func() { _ = database.Close() }()

	orch := orchestrator.NewOrchestrator("./worker", 8090)
	scheduler := orchestrator.NewScheduler(database, orch)
	notifier := notify.NewManager(database)
	handler := NewHandler(database, orch, scheduler, notifier)

	// Seed data
	_ = database.AddDevice(db.Device{Name: "S", Hostname: "s", SSHUser: "u", SSHPort: 22})
	_ = database.AddDevice(db.Device{Name: "T", Hostname: "t", SSHUser: "u", SSHPort: 22})
	_ = database.AddResult(1, 2, "ping", 1.2, 0, 0, 0, "")

	req, _ := http.NewRequest("GET", "/results/latest", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var results []db.Result
	if err := json.NewDecoder(rr.Body).Decode(&results); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected at least one result")
	}
}
