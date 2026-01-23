package orchestrator

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// TaskType represents the type of task to execute
type TaskType string

const (
	TaskPingAll  TaskType = "ping_all"
	TaskSpeedAll TaskType = "speed_all"
)

// TaskPriority determines execution order (higher = executed first)
type TaskPriority int

const (
	PriorityNormal TaskPriority = 0 // Scheduled tasks
	PriorityHigh   TaskPriority = 1 // Manual tasks
)

// Task represents a unit of work to be executed
type Task struct {
	ID        string       `json:"id"`
	Type      TaskType     `json:"type"`
	Priority  TaskPriority `json:"priority"`
	CreatedAt time.Time    `json:"created_at"`
}

// QueueStatus provides visibility into the queue state
type QueueStatus struct {
	Running *Task  `json:"running"`
	Queued  []Task `json:"queued"`
	Length  int    `json:"length"`
}

// TaskQueue manages task execution with priority ordering
type TaskQueue struct {
	tasks    []Task
	mu       sync.Mutex
	cond     *sync.Cond
	running  *Task
	stopChan chan struct{}
	stopped  bool

	OnStatus func(string)
}

// NewTaskQueue creates a new task queue
func NewTaskQueue() *TaskQueue {
	q := &TaskQueue{
		tasks:    make([]Task, 0),
		stopChan: make(chan struct{}),
	}
	q.cond = sync.NewCond(&q.mu)
	return q
}

// Enqueue adds a task to the queue
// High priority tasks are inserted before normal priority tasks
func (q *TaskQueue) Enqueue(t Task) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.stopped {
		return
	}

	// Generate ID if not set
	if t.ID == "" {
		t.ID = uuid.New().String()[:8]
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now()
	}

	// Check for duplicate task type already in queue or running
	if q.running != nil && q.running.Type == t.Type {
		// Same type already running, only enqueue if higher priority
		if t.Priority <= q.running.Priority {
			return
		}
	}

	// Check if same type already queued
	for _, existing := range q.tasks {
		if existing.Type == t.Type {
			// Same type already queued, only keep if this one is higher priority
			if t.Priority <= existing.Priority {
				return
			}
			// Remove the lower priority one
			q.removeTaskLocked(existing.ID)
			break
		}
	}

	// Insert based on priority (high priority first)
	inserted := false
	for i, existing := range q.tasks {
		if t.Priority > existing.Priority {
			// Insert before this task
			q.tasks = append(q.tasks[:i], append([]Task{t}, q.tasks[i:]...)...)
			inserted = true
			break
		}
	}
	if !inserted {
		q.tasks = append(q.tasks, t)
	}

	q.cond.Signal()
}

// removeTaskLocked removes a task by ID (must hold lock)
func (q *TaskQueue) removeTaskLocked(id string) {
	for i, t := range q.tasks {
		if t.ID == id {
			q.tasks = append(q.tasks[:i], q.tasks[i+1:]...)
			return
		}
	}
}

// Start begins processing tasks with the given executor function
func (q *TaskQueue) Start(executor func(Task)) {
	go func() {
		for {
			q.mu.Lock()
			// Wait for task or stop signal
			for len(q.tasks) == 0 && !q.stopped {
				q.cond.Wait()
			}

			if q.stopped {
				q.mu.Unlock()
				return
			}

			// Get next task
			task := q.tasks[0]
			q.tasks = q.tasks[1:]
			q.running = &task
			q.mu.Unlock()

			// Execute task
			executor(task)

			// Clear running state
			q.mu.Lock()
			q.running = nil
			q.mu.Unlock()
		}
	}()
}

// Stop signals the queue to stop processing
func (q *TaskQueue) Stop() {
	q.mu.Lock()
	q.stopped = true
	q.mu.Unlock()
	q.cond.Signal()
}

// GetStatus returns current queue status
func (q *TaskQueue) GetStatus() QueueStatus {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Make a copy of queued tasks
	queued := make([]Task, len(q.tasks))
	copy(queued, q.tasks)

	var running *Task
	if q.running != nil {
		r := *q.running
		running = &r
	}

	return QueueStatus{
		Running: running,
		Queued:  queued,
		Length:  len(q.tasks),
	}
}

// IsRunning returns true if a task is currently being executed
func (q *TaskQueue) IsRunning() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.running != nil
}
