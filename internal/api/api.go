package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/user/homelab-speedtest/internal/db"
	"github.com/user/homelab-speedtest/internal/notify"
	"github.com/user/homelab-speedtest/internal/orchestrator"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

type Handler struct {
	*http.ServeMux
	db        *db.DB
	orch      *orchestrator.Orchestrator
	scheduler *orchestrator.Scheduler
	notifier  *notify.Manager

	// SSE clients
	clientsMu sync.Mutex
	clients   map[chan any]bool

	// WebSocket clients
	wsClientsMu sync.RWMutex
	wsClients   map[*websocket.Conn]bool
}

func NewHandler(d *db.DB, orch *orchestrator.Orchestrator, scheduler *orchestrator.Scheduler, notifier *notify.Manager) *Handler {
	h := &Handler{
		ServeMux:  http.NewServeMux(),
		db:        d,
		orch:      orch,
		scheduler: scheduler,
		notifier:  notifier,
		clients:   make(map[chan any]bool),
		wsClients: make(map[*websocket.Conn]bool),
	}
	h.routes()
	return h
}

func (h *Handler) BroadcastResult(res db.Result) {
	h.broadcast(map[string]any{
		"type": "result",
		"data": res,
	})
}

func (h *Handler) BroadcastStatus(msg string) {
	h.broadcast(map[string]any{
		"type": "status",
		"data": msg,
	})
}

func (h *Handler) BroadcastScheduleInfo(info []orchestrator.ScheduleInfo) {
	h.broadcast(map[string]any{
		"type": "schedule",
		"data": info,
	})
}

func (h *Handler) BroadcastQueueStatus(status orchestrator.QueueStatus) {
	h.broadcast(map[string]any{
		"type": "queue",
		"data": status,
	})
}

func (h *Handler) broadcast(event any) {
	// Broadcast to SSE clients
	h.clientsMu.Lock()
	for clientChan := range h.clients {
		select {
		case clientChan <- event:
		default:
		}
	}
	h.clientsMu.Unlock()

	// Broadcast to WebSocket clients
	h.wsClientsMu.RLock()
	defer h.wsClientsMu.RUnlock()

	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	for conn := range h.wsClients {
		err := conn.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			log.Printf("WebSocket write error: %v", err)
			// Don't remove here to avoid modifying map during iteration
			// Client will be removed when the read loop detects the error
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

	h.HandleFunc("/schedule-status", func(w http.ResponseWriter, r *http.Request) {
		info := h.scheduler.GetScheduleInfo()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(info)
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

		clientChan := make(chan any, 10)
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
				_, _ = fmt.Fprintf(w, ": keep-alive\n\n")
				flusher.Flush()
			case event := <-clientChan:
				data, err := json.Marshal(event)
				if err != nil {
					continue
				}
				_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			}
		}
	})

	h.HandleFunc("POST /test/ping/all", func(w http.ResponseWriter, r *http.Request) {
		go h.scheduler.RunAllPings()
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status": "initiated"}`))
	})

	h.HandleFunc("POST /test/speed/all", func(w http.ResponseWriter, r *http.Request) {
		go h.scheduler.RunAllSpeeds()
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status": "initiated"}`))
	})

	// Queue status endpoint
	h.HandleFunc("/queue-status", func(w http.ResponseWriter, r *http.Request) {
		status := h.scheduler.GetQueueStatus()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(status)
	})

	// Notification settings endpoints
	h.HandleFunc("/notification-settings", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			if h.notifier == nil {
				http.Error(w, "Notifications not configured", http.StatusServiceUnavailable)
				return
			}
			settings := h.notifier.GetSettings()
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(settings)
		case "PUT":
			if h.notifier == nil {
				http.Error(w, "Notifications not configured", http.StatusServiceUnavailable)
				return
			}
			var settings notify.NotificationSettings
			if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			if err := h.notifier.UpdateSettings(settings); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Alert rules endpoints
	h.HandleFunc("/alert-rules", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			rules, err := h.db.GetAlertRules()
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(rules)
		case "POST":
			var rule db.AlertRule
			if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			id, err := h.db.CreateAlertRule(rule)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]int64{"id": id})
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	h.HandleFunc("DELETE /alert-rules/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}

		if err := h.db.DeleteAlertRule(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	h.HandleFunc("PUT /alert-rules/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}

		var rule db.AlertRule
		if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		rule.ID = id

		if err := h.db.UpdateAlertRule(rule); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	// WebSocket endpoint for real-time updates
	h.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("WebSocket upgrade error: %v", err)
			return
		}

		// Register client
		h.wsClientsMu.Lock()
		h.wsClients[conn] = true
		h.wsClientsMu.Unlock()

		log.Printf("WebSocket client connected. Total: %d", len(h.wsClients))

		// Send initial schedule status
		info := h.scheduler.GetScheduleInfo()
		initialMsg, _ := json.Marshal(map[string]any{
			"type": "schedule",
			"data": info,
		})
		_ = conn.WriteMessage(websocket.TextMessage, initialMsg)

		// Cleanup on disconnect
		defer func() {
			h.wsClientsMu.Lock()
			delete(h.wsClients, conn)
			h.wsClientsMu.Unlock()
			_ = conn.Close()
			log.Printf("WebSocket client disconnected. Total: %d", len(h.wsClients))
		}()

		// Read loop to detect disconnects and handle pings
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				}
				break
			}
		}
	})
}
