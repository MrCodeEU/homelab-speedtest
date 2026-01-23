package db

import (
	"database/sql"
	_ "embed"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/user/homelab-speedtest/internal/config"
)

//go:embed schema.sql
var schema string

type DB struct {
	*sql.DB
}

func New(cfg config.DatabaseConfig) (*DB, error) {
	// Add busy_timeout to handle concurrent writes
	dsn := cfg.Path + "?_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("failed to apply schema: %w", err)
	}

	// Simple migration: check if error column exists, if not add it
	// (This is a quick fix for existing DBs)
	// We ignore the error because if it exists, it's fine.
	_, _ = db.Exec("ALTER TABLE results ADD COLUMN error TEXT")

	return &DB{DB: db}, nil
}

type Device struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Hostname string `json:"hostname"`
	IP       string `json:"ip"` // Added IP
	SSHUser  string `json:"ssh_user"`
	SSHPort  int    `json:"ssh_port"`
}

func (d *DB) GetDevices() ([]Device, error) {
	rows, err := d.Query("SELECT id, name, hostname, IFNULL(ip, ''), ssh_user, ssh_port FROM devices")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	devices := []Device{}
	for rows.Next() {
		var dev Device
		if err := rows.Scan(&dev.ID, &dev.Name, &dev.Hostname, &dev.IP, &dev.SSHUser, &dev.SSHPort); err != nil {
			return nil, err
		}
		devices = append(devices, dev)
	}
	return devices, nil
}

func (d *DB) AddDevice(dev Device) error {
	_, err := d.Exec("INSERT INTO devices (name, hostname, ip, ssh_user, ssh_port) VALUES (?, ?, ?, ?, ?)",
		dev.Name, dev.Hostname, dev.IP, dev.SSHUser, dev.SSHPort)
	return err
}

func (d *DB) DeleteDevice(id int) error {
	_, err := d.Exec("DELETE FROM devices WHERE id = ?", id)
	return err
}

func (d *DB) AddResult(sourceID, targetID int, type_ string, latency, jitter, loss, bandwidth float64, errorMsg string) error {
	_, err := d.Exec(`INSERT INTO results 
		(source_device_id, target_device_id, type, latency_ms, jitter_ms, packet_loss, bandwidth_mbps, error) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		sourceID, targetID, type_, latency, jitter, loss, bandwidth, errorMsg)
	return err
}

type Schedule struct {
	ID      int    `json:"id"`
	Type    string `json:"type"` // 'ping' or 'speed'
	Cron    string `json:"cron"`
	Enabled bool   `json:"enabled"`
}

func (d *DB) GetSchedules() ([]Schedule, error) {
	rows, err := d.Query("SELECT id, type, cron, enabled FROM schedules")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	schedules := []Schedule{}
	for rows.Next() {
		var s Schedule
		if err := rows.Scan(&s.ID, &s.Type, &s.Cron, &s.Enabled); err != nil {
			return nil, err
		}
		schedules = append(schedules, s)
	}
	return schedules, nil
}

func (d *DB) UpdateSchedule(type_ string, cron string, enabled bool) error {
	// Upsert based on type
	// SQLite upsert syntax: INSERT INTO ... ON CONFLICT(type) DO UPDATE SET ...
	// Assuming 'type' is unique or we just update all of that type?
	// The schema has specific ID. Let's assume we maintain one row per type for global config for now.
	// Or check if it exists.
	// Actually schema.sql says: id INTEGER PRIMARY KEY, type TEXT. It doesn't enforce unique type.
	// But our seed.sql inserts one for 'ping' and one for 'speed'.
	// Let's UPDATE based on type.

	res, err := d.Exec("UPDATE schedules SET cron = ?, enabled = ? WHERE type = ?", cron, enabled, type_)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		_, err = d.Exec("INSERT INTO schedules (type, cron, enabled) VALUES (?, ?, ?)", type_, cron, enabled)
	}
	return err
}

func (d *DB) GetHistory(limit int) ([]Result, error) {
	query := `
		SELECT 
			source_device_id, 
			target_device_id, 
			type, 
			IFNULL(latency_ms, 0), 
			IFNULL(bandwidth_mbps, 0), 
			timestamp,
			IFNULL(error, '')
		FROM results 
		ORDER BY timestamp DESC 
		LIMIT ?
	`
	rows, err := d.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	results := []Result{}
	for rows.Next() {
		var res Result
		if err := rows.Scan(&res.SourceID, &res.TargetID, &res.Type, &res.LatencyMs, &res.BandwidthMbps, &res.Timestamp, &res.Error); err != nil {
			return nil, err
		}
		res.Error = strings.TrimSpace(res.Error)
		results = append(results, res)
	}
	return results, nil
}

type Result struct {
	SourceID      int     `json:"source_id"`
	TargetID      int     `json:"target_id"`
	Type          string  `json:"type"`
	LatencyMs     float64 `json:"latency_ms"`
	JitterMs      float64 `json:"jitter_ms"`
	PacketLoss    float64 `json:"packet_loss"`
	BandwidthMbps float64 `json:"bandwidth_mbps"`
	Timestamp     string  `json:"timestamp"`
	Error         string  `json:"error"`
}

// Notification Settings

func (d *DB) GetNotificationSetting(key string) (string, error) {
	var value string
	err := d.QueryRow("SELECT value FROM notification_settings WHERE key = ?", key).Scan(&value)
	if err != nil {
		return "", err
	}
	return value, nil
}

func (d *DB) SetNotificationSetting(key, value string) error {
	_, err := d.Exec(`INSERT INTO notification_settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

func (d *DB) GetAllNotificationSettings() (map[string]string, error) {
	rows, err := d.Query("SELECT key, value FROM notification_settings")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	settings := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		settings[key] = value
	}
	return settings, nil
}

// Alert Rules

type AlertRule struct {
	ID              int      `json:"id"`
	Name            string   `json:"name"`
	EventType       string   `json:"event_type"` // 'speed_below', 'ping_above', 'packet_loss_above', 'test_error'
	Threshold       *float64 `json:"threshold"`  // NULL for test_error
	SourceDeviceID  *int     `json:"source_device_id"`
	TargetDeviceID  *int     `json:"target_device_id"`
	NotifyNtfy      bool     `json:"notify_ntfy"`
	NtfyTopic       string   `json:"ntfy_topic"`
	NotifyEmail     bool     `json:"notify_email"`
	EmailRecipients string   `json:"email_recipients"`
	Enabled         bool     `json:"enabled"`
	CreatedAt       string   `json:"created_at"`
}

func (d *DB) GetAlertRules() ([]AlertRule, error) {
	rows, err := d.Query(`SELECT id, name, event_type, threshold, source_device_id, target_device_id,
		notify_ntfy, IFNULL(ntfy_topic, ''), notify_email, IFNULL(email_recipients, ''), enabled, created_at
		FROM alert_rules ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	rules := []AlertRule{}
	for rows.Next() {
		var r AlertRule
		if err := rows.Scan(&r.ID, &r.Name, &r.EventType, &r.Threshold, &r.SourceDeviceID, &r.TargetDeviceID,
			&r.NotifyNtfy, &r.NtfyTopic, &r.NotifyEmail, &r.EmailRecipients, &r.Enabled, &r.CreatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}
	return rules, nil
}

func (d *DB) CreateAlertRule(rule AlertRule) (int64, error) {
	res, err := d.Exec(`INSERT INTO alert_rules
		(name, event_type, threshold, source_device_id, target_device_id, notify_ntfy, ntfy_topic, notify_email, email_recipients, enabled)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rule.Name, rule.EventType, rule.Threshold, rule.SourceDeviceID, rule.TargetDeviceID,
		rule.NotifyNtfy, rule.NtfyTopic, rule.NotifyEmail, rule.EmailRecipients, rule.Enabled)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DB) UpdateAlertRule(rule AlertRule) error {
	_, err := d.Exec(`UPDATE alert_rules SET
		name = ?, event_type = ?, threshold = ?, source_device_id = ?, target_device_id = ?,
		notify_ntfy = ?, ntfy_topic = ?, notify_email = ?, email_recipients = ?, enabled = ?
		WHERE id = ?`,
		rule.Name, rule.EventType, rule.Threshold, rule.SourceDeviceID, rule.TargetDeviceID,
		rule.NotifyNtfy, rule.NtfyTopic, rule.NotifyEmail, rule.EmailRecipients, rule.Enabled, rule.ID)
	return err
}

func (d *DB) DeleteAlertRule(id int) error {
	_, err := d.Exec("DELETE FROM alert_rules WHERE id = ?", id)
	return err
}

func (d *DB) GetLatestResults() ([]Result, error) {
	query := `
		SELECT 
			r.source_device_id, 
			r.target_device_id, 
			r.type, 
			IFNULL(r.latency_ms, 0), 
			IFNULL(r.bandwidth_mbps, 0), 
			r.timestamp,
			IFNULL(r.error, '')
		FROM results r
		INNER JOIN (
			SELECT source_device_id, target_device_id, type, MAX(timestamp) as max_ts
			FROM results
			GROUP BY source_device_id, target_device_id, type
		) latest ON r.source_device_id = latest.source_device_id 
				AND r.target_device_id = latest.target_device_id 
				AND r.type = latest.type 
				AND r.timestamp = latest.max_ts
	`
	rows, err := d.Query(query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	results := []Result{}
	for rows.Next() {
		var res Result
		if err := rows.Scan(&res.SourceID, &res.TargetID, &res.Type, &res.LatencyMs, &res.BandwidthMbps, &res.Timestamp, &res.Error); err != nil {
			return nil, err
		}
		// If error is present, we might want to trim it or just pass it through
		res.Error = strings.TrimSpace(res.Error)
		results = append(results, res)
	}
	return results, nil
}
