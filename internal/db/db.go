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
	db, err := sql.Open("sqlite", cfg.Path)
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

type Result struct {
	SourceID      int     `json:"source_id"`
	TargetID      int     `json:"target_id"`
	Type          string  `json:"type"`
	LatencyMs     float64 `json:"latency_ms"`
	BandwidthMbps float64 `json:"bandwidth_mbps"`
	Timestamp     string  `json:"timestamp"`
	Error         string  `json:"error"`
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
