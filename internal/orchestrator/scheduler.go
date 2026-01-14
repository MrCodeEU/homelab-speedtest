package orchestrator

import (
	"log"
	"time"

	"github.com/user/homelab-speedtest/internal/db"
)

type Scheduler struct {
	db   *db.DB
	orch *Orchestrator

	// Configurable intervals
	PingInterval  time.Duration
	SpeedInterval time.Duration
}

func NewScheduler(d *db.DB, orch *Orchestrator) *Scheduler {
	// Defaults
	return &Scheduler{
		db:            d,
		orch:          orch,
		PingInterval:  1 * time.Minute,
		SpeedInterval: 15 * time.Minute,
	}
}

func (s *Scheduler) Start() {
	go func() {
		pingTicker := time.NewTicker(s.PingInterval)
		speedTicker := time.NewTicker(s.SpeedInterval)

		for {
			select {
			case <-pingTicker.C:
				s.RunAllPings()
			case <-speedTicker.C:
				s.RunAllSpeeds()
			}
		}
	}()
	log.Println("Scheduler started")
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
			go func(src, dst db.Device) {
				resp, err := s.orch.RunPing(src, dst)
				if err != nil {
					log.Printf("Ping %s->%s failed: %v", src.Name, dst.Name, err)
					return
				}
				// Save result
				if err := s.db.AddResult(src.ID, dst.ID, "ping", resp.LatencyMs, resp.JitterMs, resp.PacketLoss, 0); err != nil {
					log.Printf("Failed to save result: %v", err)
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
			go func(src, dst db.Device) {
				resp, err := s.orch.RunSpeedTest(src, dst)
				if err != nil {
					log.Printf("Speed %s->%s failed: %v", src.Name, dst.Name, err)
					return
				}
				// Save result
				if err := s.db.AddResult(src.ID, dst.ID, "speed", 0, 0, 0, resp.BandwidthMbps); err != nil {
					log.Printf("Failed to save result: %v", err)
				}
			}(source, target)
		}
	}
}
