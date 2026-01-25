package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/user/homelab-speedtest/internal/api"
	"github.com/user/homelab-speedtest/internal/config"
	"github.com/user/homelab-speedtest/internal/db"
	"github.com/user/homelab-speedtest/internal/notify"
	"github.com/user/homelab-speedtest/internal/orchestrator"
)

func main() {
	// configPath := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// 1. Load Config from environment variables with defaults
	serverPort := 8080
	if portStr := os.Getenv("SERVER_PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil && p > 0 {
			serverPort = p
		}
	}

	dbPath := "data/speedtest.db"
	if p := os.Getenv("DATABASE_PATH"); p != "" {
		dbPath = p
	}

	workerPort := 8090
	if portStr := os.Getenv("WORKER_PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil && p > 0 {
			workerPort = p
		}
	}

	cfg := config.Config{
		Server:   config.ServerConfig{Port: serverPort},
		Database: config.DatabaseConfig{Path: dbPath},
	}

	// 2. Init DB
	if err := os.MkdirAll(filepath.Dir(cfg.Database.Path), 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}
	database, err := db.New(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to init db: %v", err)
	}

	// Seed default schedules if none exist
	seedDefaultSchedules(database)

	// 3. Init Orchestrator
	// Assume worker binary is in current dir or specific path
	workerPath := "./worker"
	orch := orchestrator.NewOrchestrator(workerPath, workerPort)
	log.Printf("Worker port configured: %d", workerPort)

	// 4. Init Notification Manager
	notifier := notify.NewManager(database)

	// 5. Init Scheduler
	scheduler := orchestrator.NewScheduler(database, orch)
	scheduler.Start()

	// 6. Init API
	apiHandler := api.NewHandler(database, orch, scheduler, notifier)

	// Wire up callbacks
	scheduler.OnResult = func(result db.Result) {
		apiHandler.BroadcastResult(result)
		// Check alert rules and send notifications
		devices, _ := database.GetDevices()
		notifier.CheckAndNotify(result, devices)
	}
	scheduler.OnStatus = apiHandler.BroadcastStatus
	scheduler.OnScheduleInfo = apiHandler.BroadcastScheduleInfo
	scheduler.OnQueueStatus = apiHandler.BroadcastQueueStatus

	// 5. Start Server
	// Serve UI static files (built from Svelte) at /
	// API at /api

	http.Handle("/api/", http.StripPrefix("/api", apiHandler))

	// Create a file server for the Svelte build (usually 'ui/build' or 'ui/dist')
	// For production, we'd embed this. For dev, we might proxy.
	// We'll assume the user builds the UI to 'ui/build'
	fs := http.FileServer(http.Dir("./ui/build"))
	http.Handle("/", fs)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("Server listening on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// seedDefaultSchedules creates default ping and speed schedules if none exist.
// Can be overridden with PING_SCHEDULE and SPEEDTEST_SCHEDULE environment variables.
func seedDefaultSchedules(database *db.DB) {
	schedules, err := database.GetSchedules()
	if err != nil {
		log.Printf("Warning: failed to check existing schedules: %v", err)
		return
	}

	// Check which schedule types already exist
	hasPing := false
	hasSpeed := false
	for _, s := range schedules {
		if s.Type == "ping" {
			hasPing = true
		}
		if s.Type == "speed" {
			hasSpeed = true
		}
	}

	// Create ping schedule if not exists
	if !hasPing {
		pingSchedule := os.Getenv("PING_SCHEDULE")
		if pingSchedule == "" {
			pingSchedule = "1m" // default: every 1 minute
		}
		if err := database.UpdateSchedule("ping", pingSchedule, true); err != nil {
			log.Printf("Warning: failed to create default ping schedule: %v", err)
		} else {
			log.Printf("Created default ping schedule: %s", pingSchedule)
		}
	}

	// Create speed schedule if not exists
	if !hasSpeed {
		speedSchedule := os.Getenv("SPEEDTEST_SCHEDULE")
		if speedSchedule == "" {
			speedSchedule = "15m" // default: every 15 minutes
		}
		if err := database.UpdateSchedule("speed", speedSchedule, true); err != nil {
			log.Printf("Warning: failed to create default speed schedule: %v", err)
		} else {
			log.Printf("Created default speed schedule: %s", speedSchedule)
		}
	}
}
