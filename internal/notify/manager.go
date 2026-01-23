package notify

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/user/homelab-speedtest/internal/config"
	"github.com/user/homelab-speedtest/internal/db"
)

// Event types for alert rules
const (
	EventSpeedBelow      = "speed_below"
	EventPingAbove       = "ping_above"
	EventPacketLossAbove = "packet_loss_above"
	EventTestError       = "test_error"
)

// EnvConfigStatus indicates which settings are configured via environment variables
type EnvConfigStatus struct {
	NtfyEnabled  bool `json:"ntfy_enabled"`
	NtfyServer   bool `json:"ntfy_server"`
	NtfyTopic    bool `json:"ntfy_topic"`
	NtfyToken    bool `json:"ntfy_token"`
	SMTPEnabled  bool `json:"smtp_enabled"`
	SMTPHost     bool `json:"smtp_host"`
	SMTPPort     bool `json:"smtp_port"`
	SMTPUser     bool `json:"smtp_user"`
	SMTPPassword bool `json:"smtp_password"`
	SMTPFrom     bool `json:"smtp_from"`
}

// NotificationSettings represents all notification configuration
type NotificationSettings struct {
	Ntfy          NtfySettings    `json:"ntfy"`
	SMTP          SMTPSettings    `json:"smtp"`
	EnvConfigured EnvConfigStatus `json:"env_configured"`
}

// NtfySettings for ntfy configuration
type NtfySettings struct {
	Enabled bool   `json:"enabled"`
	Server  string `json:"server"`
	Topic   string `json:"topic"`
	Token   string `json:"token"`
}

// SMTPSettings for SMTP configuration
type SMTPSettings struct {
	Enabled  bool   `json:"enabled"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	From     string `json:"from"`
}

// Manager orchestrates notifications based on alert rules
type Manager struct {
	db         *db.DB
	ntfy       *NtfyService
	email      *EmailService
	envConfig  EnvConfigStatus
	ntfyConfig config.NtfyConfig
	smtpConfig SMTPConfig
}

// NewManager creates a new notification manager
func NewManager(database *db.DB) *Manager {
	m := &Manager{
		db: database,
	}

	// Load configuration from environment variables
	m.loadEnvConfig()

	// Initialize services
	m.ntfy = New(m.ntfyConfig)
	m.email = NewEmailService(m.smtpConfig)

	return m
}

// loadEnvConfig loads configuration from environment variables
func (m *Manager) loadEnvConfig() {
	// ntfy configuration
	if v := os.Getenv("NTFY_ENABLED"); v != "" {
		m.ntfyConfig.Enabled = v == "true" || v == "1"
		m.envConfig.NtfyEnabled = true
	}
	if v := os.Getenv("NTFY_SERVER"); v != "" {
		m.ntfyConfig.Server = v
		m.envConfig.NtfyServer = true
	} else {
		m.ntfyConfig.Server = "https://ntfy.sh"
	}
	if v := os.Getenv("NTFY_TOPIC"); v != "" {
		m.ntfyConfig.Topic = v
		m.envConfig.NtfyTopic = true
	}
	if v := os.Getenv("NTFY_TOKEN"); v != "" {
		m.ntfyConfig.Token = v
		m.envConfig.NtfyToken = true
	}

	// SMTP configuration
	if v := os.Getenv("SMTP_ENABLED"); v != "" {
		m.smtpConfig.Enabled = v == "true" || v == "1"
		m.envConfig.SMTPEnabled = true
	}
	if v := os.Getenv("SMTP_HOST"); v != "" {
		m.smtpConfig.Host = v
		m.envConfig.SMTPHost = true
	}
	if v := os.Getenv("SMTP_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			m.smtpConfig.Port = port
		}
		m.envConfig.SMTPPort = true
	} else {
		m.smtpConfig.Port = 587
	}
	if v := os.Getenv("SMTP_USER"); v != "" {
		m.smtpConfig.User = v
		m.envConfig.SMTPUser = true
	}
	if v := os.Getenv("SMTP_PASSWORD"); v != "" {
		m.smtpConfig.Password = v
		m.envConfig.SMTPPassword = true
	}
	if v := os.Getenv("SMTP_FROM"); v != "" {
		m.smtpConfig.From = v
		m.envConfig.SMTPFrom = true
	}

	// Load DB settings for any not set via env
	m.loadDBSettings()
}

// loadDBSettings loads settings from database for values not set via env
func (m *Manager) loadDBSettings() {
	settings, err := m.db.GetAllNotificationSettings()
	if err != nil {
		log.Printf("Failed to load notification settings from DB: %v", err)
		return
	}

	// ntfy settings from DB
	if !m.envConfig.NtfyEnabled {
		if v, ok := settings["ntfy_enabled"]; ok {
			m.ntfyConfig.Enabled = v == "true" || v == "1"
		}
	}
	if !m.envConfig.NtfyServer {
		if v, ok := settings["ntfy_server"]; ok && v != "" {
			m.ntfyConfig.Server = v
		}
	}
	if !m.envConfig.NtfyTopic {
		if v, ok := settings["ntfy_topic"]; ok {
			m.ntfyConfig.Topic = v
		}
	}
	if !m.envConfig.NtfyToken {
		if v, ok := settings["ntfy_token"]; ok {
			m.ntfyConfig.Token = v
		}
	}

	// SMTP settings from DB
	if !m.envConfig.SMTPEnabled {
		if v, ok := settings["smtp_enabled"]; ok {
			m.smtpConfig.Enabled = v == "true" || v == "1"
		}
	}
	if !m.envConfig.SMTPHost {
		if v, ok := settings["smtp_host"]; ok {
			m.smtpConfig.Host = v
		}
	}
	if !m.envConfig.SMTPPort {
		if v, ok := settings["smtp_port"]; ok {
			if port, err := strconv.Atoi(v); err == nil {
				m.smtpConfig.Port = port
			}
		}
	}
	if !m.envConfig.SMTPUser {
		if v, ok := settings["smtp_user"]; ok {
			m.smtpConfig.User = v
		}
	}
	if !m.envConfig.SMTPPassword {
		if v, ok := settings["smtp_password"]; ok {
			m.smtpConfig.Password = v
		}
	}
	if !m.envConfig.SMTPFrom {
		if v, ok := settings["smtp_from"]; ok {
			m.smtpConfig.From = v
		}
	}

	// Reinitialize services with updated config
	m.ntfy = New(m.ntfyConfig)
	m.email = NewEmailService(m.smtpConfig)
}

// GetSettings returns current notification settings
func (m *Manager) GetSettings() NotificationSettings {
	return NotificationSettings{
		Ntfy: NtfySettings{
			Enabled: m.ntfyConfig.Enabled,
			Server:  m.ntfyConfig.Server,
			Topic:   m.ntfyConfig.Topic,
			Token:   m.ntfyConfig.Token,
		},
		SMTP: SMTPSettings{
			Enabled:  m.smtpConfig.Enabled,
			Host:     m.smtpConfig.Host,
			Port:     m.smtpConfig.Port,
			User:     m.smtpConfig.User,
			Password: m.smtpConfig.Password,
			From:     m.smtpConfig.From,
		},
		EnvConfigured: m.envConfig,
	}
}

// UpdateSettings updates notification settings in the database
func (m *Manager) UpdateSettings(s NotificationSettings) error {
	// Only update settings not configured via env
	if !m.envConfig.NtfyEnabled {
		if err := m.db.SetNotificationSetting("ntfy_enabled", fmt.Sprintf("%v", s.Ntfy.Enabled)); err != nil {
			return err
		}
		m.ntfyConfig.Enabled = s.Ntfy.Enabled
	}
	if !m.envConfig.NtfyServer {
		if err := m.db.SetNotificationSetting("ntfy_server", s.Ntfy.Server); err != nil {
			return err
		}
		m.ntfyConfig.Server = s.Ntfy.Server
	}
	if !m.envConfig.NtfyTopic {
		if err := m.db.SetNotificationSetting("ntfy_topic", s.Ntfy.Topic); err != nil {
			return err
		}
		m.ntfyConfig.Topic = s.Ntfy.Topic
	}
	if !m.envConfig.NtfyToken {
		if err := m.db.SetNotificationSetting("ntfy_token", s.Ntfy.Token); err != nil {
			return err
		}
		m.ntfyConfig.Token = s.Ntfy.Token
	}

	// SMTP settings
	if !m.envConfig.SMTPEnabled {
		if err := m.db.SetNotificationSetting("smtp_enabled", fmt.Sprintf("%v", s.SMTP.Enabled)); err != nil {
			return err
		}
		m.smtpConfig.Enabled = s.SMTP.Enabled
	}
	if !m.envConfig.SMTPHost {
		if err := m.db.SetNotificationSetting("smtp_host", s.SMTP.Host); err != nil {
			return err
		}
		m.smtpConfig.Host = s.SMTP.Host
	}
	if !m.envConfig.SMTPPort {
		if err := m.db.SetNotificationSetting("smtp_port", strconv.Itoa(s.SMTP.Port)); err != nil {
			return err
		}
		m.smtpConfig.Port = s.SMTP.Port
	}
	if !m.envConfig.SMTPUser {
		if err := m.db.SetNotificationSetting("smtp_user", s.SMTP.User); err != nil {
			return err
		}
		m.smtpConfig.User = s.SMTP.User
	}
	if !m.envConfig.SMTPPassword {
		if err := m.db.SetNotificationSetting("smtp_password", s.SMTP.Password); err != nil {
			return err
		}
		m.smtpConfig.Password = s.SMTP.Password
	}
	if !m.envConfig.SMTPFrom {
		if err := m.db.SetNotificationSetting("smtp_from", s.SMTP.From); err != nil {
			return err
		}
		m.smtpConfig.From = s.SMTP.From
	}

	// Reinitialize services
	m.ntfy = New(m.ntfyConfig)
	m.email = NewEmailService(m.smtpConfig)

	return nil
}

// CheckAndNotify checks alert rules against a result and sends notifications
func (m *Manager) CheckAndNotify(result db.Result, devices []db.Device) {
	rules, err := m.db.GetAlertRules()
	if err != nil {
		log.Printf("Failed to get alert rules: %v", err)
		return
	}

	// Get device names for the message
	var sourceName, targetName string
	for _, d := range devices {
		if d.ID == result.SourceID {
			sourceName = d.Name
		}
		if d.ID == result.TargetID {
			targetName = d.Name
		}
	}

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		// Check if rule applies to this device pair
		if rule.SourceDeviceID != nil && *rule.SourceDeviceID != result.SourceID {
			continue
		}
		if rule.TargetDeviceID != nil && *rule.TargetDeviceID != result.TargetID {
			continue
		}

		triggered := false
		var title, message string

		switch rule.EventType {
		case EventSpeedBelow:
			if result.Type == "speed" && result.Error == "" && rule.Threshold != nil {
				if result.BandwidthMbps < *rule.Threshold {
					triggered = true
					title = fmt.Sprintf("Speed Alert: %s -> %s", sourceName, targetName)
					message = fmt.Sprintf("Bandwidth %.2f Mbps is below threshold %.2f Mbps", result.BandwidthMbps, *rule.Threshold)
				}
			}
		case EventPingAbove:
			if result.Type == "ping" && result.Error == "" && rule.Threshold != nil {
				if result.LatencyMs > *rule.Threshold {
					triggered = true
					title = fmt.Sprintf("Latency Alert: %s -> %s", sourceName, targetName)
					message = fmt.Sprintf("Latency %.2f ms is above threshold %.2f ms", result.LatencyMs, *rule.Threshold)
				}
			}
		case EventPacketLossAbove:
			if result.Type == "ping" && result.Error == "" && rule.Threshold != nil {
				if result.PacketLoss > *rule.Threshold {
					triggered = true
					title = fmt.Sprintf("Packet Loss Alert: %s -> %s", sourceName, targetName)
					message = fmt.Sprintf("Packet loss %.2f%% is above threshold %.2f%%", result.PacketLoss, *rule.Threshold)
				}
			}
		case EventTestError:
			if result.Error != "" {
				triggered = true
				title = fmt.Sprintf("Test Error: %s -> %s", sourceName, targetName)
				message = fmt.Sprintf("Test failed: %s", result.Error)
			}
		}

		if triggered {
			log.Printf("Alert triggered: %s - %s", rule.Name, message)
			m.sendNotification(rule, title, message)
		}
	}
}

// sendNotification sends notifications based on rule configuration
func (m *Manager) sendNotification(rule db.AlertRule, title, message string) {
	// Send ntfy notification
	if rule.NotifyNtfy {
		topic := rule.NtfyTopic
		if topic == "" {
			topic = m.ntfyConfig.Topic
		}
		if topic != "" {
			if err := m.ntfy.SendToTopic(topic, title, message, "high"); err != nil {
				log.Printf("Failed to send ntfy notification: %v", err)
			} else {
				log.Printf("Sent ntfy notification to topic %s", topic)
			}
		}
	}

	// Send email notification
	if rule.NotifyEmail && rule.EmailRecipients != "" {
		recipients := ParseRecipients(rule.EmailRecipients)
		if len(recipients) > 0 {
			body := fmt.Sprintf("Alert: %s\n\n%s\n\nRule: %s", title, message, rule.Name)
			if err := m.email.Send(recipients, title, body); err != nil {
				log.Printf("Failed to send email notification: %v", err)
			} else {
				log.Printf("Sent email notification to %s", strings.Join(recipients, ", "))
			}
		}
	}
}

// IsConfiguredFromEnv returns the env configuration status
func (m *Manager) IsConfiguredFromEnv() EnvConfigStatus {
	return m.envConfig
}
