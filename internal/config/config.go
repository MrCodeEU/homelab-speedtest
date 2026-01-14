package config

type Config struct {
	Server        ServerConfig       `yaml:"server"`
	Database      DatabaseConfig     `yaml:"database"`
	Tailscale     TailscaleConfig    `yaml:"tailscale"`
	Notifications NotificationConfig `yaml:"notifications"`
}

type ServerConfig struct {
	Port int `yaml:"port" default:"8080"`
}

type DatabaseConfig struct {
	Path string `yaml:"path" default:"data/speedtest.db"`
}

type TailscaleConfig struct {
	// If needed to authenticate with Tailscale API or similar.
	// For SSH, we rely on the host's tailscale login generally,
	// or we might need an auth key if we were running embedded tailscale (tsnet).
	// For now, assuming system tailscale usage for SSH connectivity.
	AuthKey string `yaml:"auth_key,omitempty"`
}

type NotificationConfig struct {
	Ntfy NtfyConfig `yaml:"ntfy"`
}

type NtfyConfig struct {
	Enabled bool   `yaml:"enabled"`
	Server  string `yaml:"server" default:"https://ntfy.sh"`
	Topic   string `yaml:"topic"`
	Token   string `yaml:"token,omitempty"` // For protected topics

	// Thresholds
	MinSpeedMbps float64 `yaml:"min_speed_mbps"`
	MaxPingMs    float64 `yaml:"max_ping_ms"`
}
