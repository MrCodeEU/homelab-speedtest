package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/user/homelab-speedtest/internal/db"
	"github.com/user/homelab-speedtest/internal/orchestrator"
)

func NewRouter(d *db.DB, orch *orchestrator.Orchestrator) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("DELETE /devices/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}

		if err := d.DeleteDevice(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("/devices", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			devs, err := d.GetDevices()
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
			if err := d.AddDevice(dev); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			w.WriteHeader(http.StatusCreated)
		}
	})

	mux.HandleFunc("/results/latest", func(w http.ResponseWriter, r *http.Request) {
		results, err := d.GetLatestResults()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		_ = json.NewEncoder(w).Encode(results)
	})

	mux.HandleFunc("/test/speed", func(w http.ResponseWriter, r *http.Request) {
		// Trigger a speed test manually
		// Query params: source_id, target_id
		_, _ = w.Write([]byte(`{"status": "not implemented"}`))
	})

	return mux
}
