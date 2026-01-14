package orchestrator

// Mode constants
const (
	ModeServer = "server"
	ModeClient = "client"
	ModePing   = "ping"
)

// WorkerRequest is the JSON payload sent to the worker to initiate a task
type WorkerRequest struct {
	Mode   string `json:"mode"`           // server, client, ping
	Target string `json:"target"`         // For client/ping: "ip:port" or "ip"
	Port   int    `json:"port,omitempty"` // For server: port to listen on

	// Options
	DurationSeconds int `json:"duration,omitempty"`
}

// WorkerResponse is the JSON output from the worker
type WorkerResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`

	// Result Data
	LatencyMs     float64 `json:"latency_ms,omitempty"`
	JitterMs      float64 `json:"jitter_ms,omitempty"`
	PacketLoss    float64 `json:"packet_loss,omitempty"`
	BandwidthMbps float64 `json:"bandwidth_mbps,omitempty"`
}
