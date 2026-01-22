package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/user/homelab-speedtest/internal/db"
	"github.com/user/homelab-speedtest/internal/orchestrator"
)

type Handler struct {
	*http.ServeMux
	db        *db.DB
	orch      *orchestrator.Orchestrator
	scheduler *orchestrator.Scheduler

	clientsMu sync.Mutex
	clients   map[chan db.Result]bool
}

func NewHandler(d *db.DB, orch *orchestrator.Orchestrator, scheduler *orchestrator.Scheduler) *Handler {
	h := &Handler{
		ServeMux:  http.NewServeMux(),
		db:        d,
		orch:      orch,
		scheduler: scheduler,
		clients:   make(map[chan db.Result]bool),
	}
	h.routes()
	return h
}

func (h *Handler) BroadcastResult(res db.Result) {
	h.clientsMu.Lock()
	defer h.clientsMu.Unlock()

	for clientChan := range h.clients {
		select {
		case clientChan <- res:
		default:
			// Client blocked, likely disconnected or slow. Drop?
		}
	}
}

func (h *Handler) routes() {
	h.HandleFunc("/schedules", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			schedules, err := h.db.GetSchedules()
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			_ = json.NewEncoder(w).Encode(schedules)
		case "PUT":
			var req struct {
				Type    string `json:"type"`
				Cron    string `json:"cron"`
				Enabled bool   `json:"enabled"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			if err := h.db.UpdateSchedule(req.Type, req.Cron, req.Enabled); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			// Reload scheduler
		h.scheduler.Reload()
		w.WriteHeader(http.StatusOK)
		}
	})

	h.HandleFunc("/history", func(w http.ResponseWriter, r *http.Request) {
		limitStr := r.URL.Query().Get("limit")
		limit := 100
		if limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil {
				limit = l
			}
		}

		history, err := h.db.GetHistory(limit)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		_ = json.NewEncoder(w).Encode(history)
	})

	h.HandleFunc("DELETE /devices/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}

		if err := h.db.DeleteDevice(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	h.HandleFunc("/devices", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			devs, err := h.db.GetDevices()
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			_ = json.NewEncoder(w).Encode(devs)
		case "POST":
			var dev db.Device
			if err := json.NewDecoder(r.Body).Decode(&dev); err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			if err := h.db.AddDevice(dev); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			w.WriteHeader(http.StatusCreated)
		}
	})

	h.HandleFunc("/results/latest", func(w http.ResponseWriter, r *http.Request) {
		results, err := h.db.GetLatestResults()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		_ = json.NewEncoder(w).Encode(results)
	})

	// SSE Endpoint
	h.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		// Set headers for SSE
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		clientChan := make(chan db.Result, 10)
		h.clientsMu.Lock()
		h.clients[clientChan] = true
		h.clientsMu.Unlock()

		defer func() {
			h.clientsMu.Lock()
			delete(h.clients, clientChan)
			h.clientsMu.Unlock()
			close(clientChan)
		}()

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		// Keep-alive ticker
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-r.Context().Done():
				return
			case <-ticker.C:
				fmt.Fprintf(w, ": keep-alive\n\n")
				flusher.Flush()
			case res := <-clientChan:
				data, err := json.Marshal(res)
				if err != nil {
					continue
				}
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			}
		}
	})
}