package orchestrator

import (
	"log"
	"time"

	"github.com/user/homelab-speedtest/internal/db"
)

type Scheduler struct {
	db   *db.DB
	orch *Orchestrator

	stopChan chan struct{}
	OnResult func(db.Result)
}

func NewScheduler(d *db.DB, orch *Orchestrator) *Scheduler {
	return &Scheduler{
		db:       d,
		orch:     orch,
		stopChan: make(chan struct{}),
	}
}

func (s *Scheduler) Start() {
	go s.runLoop()
	log.Println("Scheduler started")
}

func (s *Scheduler) Reload() {
	s.stopChan <- struct{}{}
	go s.runLoop()
	log.Println("Scheduler reloaded")
}

func (s *Scheduler) runLoop() {
	// Load schedules from DB
	pingDuration := 1 * time.Minute
	speedDuration := 15 * time.Minute

	schedules, err := s.db.GetSchedules()
	if err == nil {
		for _, sch := range schedules {
			if !sch.Enabled {
				continue
			}
			// Try to parse as duration first (e.g. "10s", "1m")
			d, err := time.ParseDuration(sch.Cron)
			if err != nil {
				log.Printf("Invalid duration format for %s: %s. Using default.", sch.Type, sch.Cron)
				continue
			}
			if sch.Type == "ping" {
				pingDuration = d
			} else if sch.Type == "speed" {
				speedDuration = d
			}
		}
	}

	pingTicker := time.NewTicker(pingDuration)
	speedTicker := time.NewTicker(speedDuration)
	defer pingTicker.Stop()
	defer speedTicker.Stop()

	log.Printf("Scheduler running with: Ping=%v, Speed=%v", pingDuration, speedDuration)

	for {
		select {
		case <-s.stopChan:
			return
		case <-pingTicker.C:
			s.RunAllPings()
		case <-speedTicker.C:
			s.RunAllSpeeds()
		}
	}
}

func (s *Scheduler) RunAllPings() {
	log.Println("Running Ping tests...")
	devices, err := s.db.GetDevices()
	if err != nil {
		log.Printf("Failed to get devices: %v", err)
		return
	}

	for _, source := range devices {
		for _, target := range devices {
			if source.ID == target.ID {
				continue
			}
			// Run sequentially
			func(src, dst db.Device) {
				resp, err := s.orch.RunPing(src, dst)
				var errStr string
				var lat, jit, loss float64

				if err != nil {
					log.Printf("Ping %s->%s failed: %v", src.Name, dst.Name, err)
					errStr = err.Error()
				} else {
					lat = resp.LatencyMs
					jit = resp.JitterMs
					loss = resp.PacketLoss
					log.Printf("Ping %s->%s success: %.2fms", src.Name, dst.Name, lat)
				}

				// Save result
				if err := s.db.AddResult(src.ID, dst.ID, "ping", lat, jit, loss, 0, errStr); err != nil {
					log.Printf("Failed to save result: %v", err)
				} else if s.OnResult != nil {
					s.OnResult(db.Result{
						SourceID:  src.ID,
						TargetID:  dst.ID,
						Type:      "ping",
						LatencyMs: lat,
						Timestamp: time.Now().UTC().Format("2006-01-02 15:04:05"),
						Error:     errStr,
					})
				}
			}(source, target)
		}
	}
}

func (s *Scheduler) RunAllSpeeds() {
	log.Println("Running Speed tests...")
	devices, err := s.db.GetDevices()
	if err != nil {
		log.Printf("Failed to get devices: %v", err)
		return
	}

	for _, source := range devices {
		for _, target := range devices {
			if source.ID == target.ID {
				continue
			}
			// Run sequentially
			func(src, dst db.Device) {
				resp, err := s.orch.RunSpeedTest(src, dst)
				var errStr string
				var bw float64

				if err != nil {
					log.Printf("Speed %s->%s failed: %v", src.Name, dst.Name, err)
					errStr = err.Error()
				} else {
					bw = resp.BandwidthMbps
					log.Printf("Speed %s->%s success: %.2fMbps", src.Name, dst.Name, bw)
				}

				// Save result
				if err := s.db.AddResult(src.ID, dst.ID, "speed", 0, 0, 0, bw, errStr); err != nil {
					log.Printf("Failed to save result: %v", err)
				} else if s.OnResult != nil {
					s.OnResult(db.Result{
						SourceID:      src.ID,
						TargetID:      dst.ID,
						Type:          "speed",
						BandwidthMbps: bw,
						Timestamp:     time.Now().UTC().Format("2006-01-02 15:04:05"),
						Error:         errStr,
					})
				}
			}(source, target)
		}
	}
}