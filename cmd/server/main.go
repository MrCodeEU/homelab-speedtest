package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/user/homelab-speedtest/internal/api"
	"github.com/user/homelab-speedtest/internal/config"
	"github.com/user/homelab-speedtest/internal/db"
	"github.com/user/homelab-speedtest/internal/orchestrator"
)

func main() {
	// configPath := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// 1. Load Config (Placeholder for now, just default)
	cfg := config.Config{
		Server:   config.ServerConfig{Port: 8080},
		Database: config.DatabaseConfig{Path: "data/speedtest.db"},
	}

	// 2. Init DB
	if err := os.MkdirAll(filepath.Dir(cfg.Database.Path), 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}
	database, err := db.New(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to init db: %v", err)
	}

	// 3. Init Orchestrator
	// Assume worker binary is in current dir or specific path
	workerPath := "./worker"
	orch := orchestrator.NewOrchestrator(workerPath)

	// 4. Init Scheduler
	scheduler := orchestrator.NewScheduler(database, orch)
	scheduler.Start()

	// 5. Init API
	router := api.NewRouter(database, orch)

	// 5. Start Server
	// Serve UI static files (built from Svelte) at /
	// API at /api

	http.Handle("/api/", http.StripPrefix("/api", router))

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
