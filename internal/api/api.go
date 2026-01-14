package api

import (
	"encoding/json"
	"net/http"

	"github.com/user/homelab-speedtest/internal/db"
	"github.com/user/homelab-speedtest/internal/orchestrator"
)

func NewRouter(d *db.DB, orch *orchestrator.Orchestrator) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/devices", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			devs, err := d.GetDevices()
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			json.NewEncoder(w).Encode(devs)
		} else if r.Method == "POST" {
			var dev db.Device
			if err := json.NewDecoder(r.Body).Decode(&dev); err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			if err := d.AddDevice(dev); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			w.WriteHeader(http.StatusCreated)
		}
	})

	mux.HandleFunc("/test/speed", func(w http.ResponseWriter, r *http.Request) {
		// Trigger a speed test manually
		// Query params: source_id, target_id
		w.Write([]byte(`{"status": "not implemented"}`))
	})

	return mux
}
