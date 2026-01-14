package db

import (
	"database/sql"
	_ "embed"
	"fmt"

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

	var devices []Device
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

func (d *DB) AddResult(sourceID, targetID int, type_ string, latency, jitter, loss, bandwidth float64) error {
	_, err := d.Exec(`INSERT INTO results 
		(source_device_id, target_device_id, type, latency_ms, jitter_ms, packet_loss, bandwidth_mbps) 
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		sourceID, targetID, type_, latency, jitter, loss, bandwidth)
	return err
}
