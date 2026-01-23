package orchestrator

import (
	"log"
	"time"

	"github.com/user/homelab-speedtest/internal/db"
)

type ScheduleInfo struct {
	Type     string `json:"type"`
	Interval string `json:"interval"`
	Enabled  bool   `json:"enabled"`
	NextRun  string `json:"next_run"`
}

type Scheduler struct {
	db    *db.DB
	orch  *Orchestrator
	queue *TaskQueue

	stopChan       chan struct{}
	OnResult       func(db.Result)
	OnStatus       func(string)
	OnScheduleInfo func([]ScheduleInfo)
	OnQueueStatus  func(QueueStatus)

	// Schedule tracking
	pingInterval  time.Duration
	speedInterval time.Duration
	pingEnabled   bool
	speedEnabled  bool
	nextPingRun   time.Time
	nextSpeedRun  time.Time
	scheduleMu    lockedMutex
}

// lockedMutex is a simple mutex wrapper for schedule tracking
type lockedMutex struct {
	locked bool
}

func (m *lockedMutex) RLock()   {}
func (m *lockedMutex) RUnlock() {}
func (m *lockedMutex) Lock()    {}
func (m *lockedMutex) Unlock()  {}

func NewScheduler(d *db.DB, orch *Orchestrator) *Scheduler {
	s := &Scheduler{
		db:            d,
		orch:          orch,
		stopChan:      make(chan struct{}),
		pingInterval:  1 * time.Minute,
		speedInterval: 15 * time.Minute,
		pingEnabled:   true,
		speedEnabled:  true,
	}

	// Initialize task queue
	s.queue = NewTaskQueue()
	s.queue.OnStatus = func(msg string) {
		if s.OnStatus != nil {
			s.OnStatus(msg)
		}
	}

	return s
}

func (s *Scheduler) GetScheduleInfo() []ScheduleInfo {
	now := time.Now()
	pingNext := ""
	speedNext := ""

	if s.pingEnabled && !s.nextPingRun.IsZero() {
		if s.nextPingRun.After(now) {
			pingNext = s.nextPingRun.Format(time.RFC3339)
		} else {
			pingNext = now.Add(s.pingInterval).Format(time.RFC3339)
		}
	}

	if s.speedEnabled && !s.nextSpeedRun.IsZero() {
		if s.nextSpeedRun.After(now) {
			speedNext = s.nextSpeedRun.Format(time.RFC3339)
		} else {
			speedNext = now.Add(s.speedInterval).Format(time.RFC3339)
		}
	}

	return []ScheduleInfo{
		{
			Type:     "ping",
			Interval: s.pingInterval.String(),
			Enabled:  s.pingEnabled,
			NextRun:  pingNext,
		},
		{
			Type:     "speed",
			Interval: s.speedInterval.String(),
			Enabled:  s.speedEnabled,
			NextRun:  speedNext,
		},
	}
}

func (s *Scheduler) GetQueueStatus() QueueStatus {
	return s.queue.GetStatus()
}

func (s *Scheduler) Start() {
	// Start the queue worker
	s.queue.Start(s.executeTask)
	go s.runLoop()
	log.Println("Scheduler started with task queue")
}

func (s *Scheduler) Reload() {
	// Signal stop and restart
	select {
	case s.stopChan <- struct{}{}:
	default:
	}
	go s.runLoop()
	log.Println("Scheduler reloaded")
}

func (s *Scheduler) executeTask(task Task) {
	log.Printf("Executing task: %s (id=%s, priority=%d)", task.Type, task.ID, task.Priority)

	// Broadcast queue status update
	if s.OnQueueStatus != nil {
		s.OnQueueStatus(s.queue.GetStatus())
	}

	switch task.Type {
	case TaskPingAll:
		s.runAllPingsInternal()
	case TaskSpeedAll:
		s.runAllSpeedsInternal()
	}

	// Broadcast queue status after completion
	if s.OnQueueStatus != nil {
		s.OnQueueStatus(s.queue.GetStatus())
	}
}

func (s *Scheduler) runLoop() {
	// Load schedules from DB
	pingDuration := 1 * time.Minute
	speedDuration := 15 * time.Minute
	pingEnabled := true
	speedEnabled := true

	schedules, err := s.db.GetSchedules()
	if err == nil {
		for _, sch := range schedules {
			d, parseErr := time.ParseDuration(sch.Cron)
			if parseErr != nil {
				log.Printf("Invalid duration format for %s: %s. Using default.", sch.Type, sch.Cron)
				continue
			}
			switch sch.Type {
			case "ping":
				pingDuration = d
				pingEnabled = sch.Enabled
			case "speed":
				speedDuration = d
				speedEnabled = sch.Enabled
			}
		}
	}

	// Update schedule tracking
	s.pingInterval = pingDuration
	s.speedInterval = speedDuration
	s.pingEnabled = pingEnabled
	s.speedEnabled = speedEnabled
	now := time.Now()
	if pingEnabled {
		s.nextPingRun = now.Add(pingDuration)
	}
	if speedEnabled {
		s.nextSpeedRun = now.Add(speedDuration)
	}

	// Broadcast schedule info
	if s.OnScheduleInfo != nil {
		s.OnScheduleInfo(s.GetScheduleInfo())
	}

	pingTicker := time.NewTicker(pingDuration)
	speedTicker := time.NewTicker(speedDuration)
	defer pingTicker.Stop()
	defer speedTicker.Stop()

	log.Printf("Scheduler running with: Ping=%v (enabled=%v), Speed=%v (enabled=%v)", pingDuration, pingEnabled, speedDuration, speedEnabled)

	for {
		select {
		case <-s.stopChan:
			log.Println("Scheduler loop stopping")
			return
		case <-pingTicker.C:
			if pingEnabled {
				// Update next run time
				s.nextPingRun = time.Now().Add(pingDuration)
				if s.OnScheduleInfo != nil {
					s.OnScheduleInfo(s.GetScheduleInfo())
				}
				// Enqueue with normal priority (scheduled)
				s.queue.Enqueue(Task{
					Type:     TaskPingAll,
					Priority: PriorityNormal,
				})
			}
		case <-speedTicker.C:
			if speedEnabled {
				// Update next run time
				s.nextSpeedRun = time.Now().Add(speedDuration)
				if s.OnScheduleInfo != nil {
					s.OnScheduleInfo(s.GetScheduleInfo())
				}
				// Enqueue with normal priority (scheduled)
				s.queue.Enqueue(Task{
					Type:     TaskSpeedAll,
					Priority: PriorityNormal,
				})
			}
		}
	}
}

// RunAllPings enqueues a ping test with high priority (manual trigger)
func (s *Scheduler) RunAllPings() {
	s.queue.Enqueue(Task{
		Type:     TaskPingAll,
		Priority: PriorityHigh,
	})
	log.Println("Manual ping test enqueued (high priority)")
}

// RunAllSpeeds enqueues a speed test with high priority (manual trigger)
func (s *Scheduler) RunAllSpeeds() {
	s.queue.Enqueue(Task{
		Type:     TaskSpeedAll,
		Priority: PriorityHigh,
	})
	log.Println("Manual speed test enqueued (high priority)")
}

// runAllPingsInternal executes all ping tests (called by queue worker)
func (s *Scheduler) runAllPingsInternal() {
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
				if s.OnStatus != nil {
					s.OnStatus("Pinging " + src.Name + " -> " + dst.Name)
				}
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
				}

				if s.OnResult != nil {
					s.OnResult(db.Result{
						SourceID:   src.ID,
						TargetID:   dst.ID,
						Type:       "ping",
						LatencyMs:  lat,
						JitterMs:   jit,
						PacketLoss: loss,
						Timestamp:  time.Now().UTC().Format("2006-01-02 15:04:05"),
						Error:      errStr,
					})
				}
			}(source, target)
		}
	}
	if s.OnStatus != nil {
		s.OnStatus("Idle")
	}
}

// runAllSpeedsInternal executes all speed tests (called by queue worker)
func (s *Scheduler) runAllSpeedsInternal() {
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
				if s.OnStatus != nil {
					s.OnStatus("Speed Test " + src.Name + " -> " + dst.Name)
				}
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
				}

				if s.OnResult != nil {
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
	if s.OnStatus != nil {
		s.OnStatus("Idle")
	}
}
