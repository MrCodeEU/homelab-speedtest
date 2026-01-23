CREATE TABLE IF NOT EXISTS devices (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    hostname TEXT NOT NULL,
    ip TEXT,
    ssh_user TEXT NOT NULL,
    ssh_port INTEGER DEFAULT 22,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS schedules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type TEXT NOT NULL UNIQUE, -- 'ping', 'speed'
    cron TEXT NOT NULL,
    enabled BOOLEAN DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_device_id INTEGER NOT NULL,
    target_device_id INTEGER NOT NULL,
    type TEXT NOT NULL, -- 'ping', 'speed'
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Ping specific
    latency_ms REAL,
    jitter_ms REAL,
    packet_loss REAL,

    -- Speed specific
    bandwidth_mbps REAL,

    -- Error reporting
    error TEXT,
    
    FOREIGN KEY(source_device_id) REFERENCES devices(id),
    FOREIGN KEY(target_device_id) REFERENCES devices(id)
);

CREATE INDEX IF NOT EXISTS idx_results_timestamp ON results(timestamp);
CREATE INDEX IF NOT EXISTS idx_results_source ON results(source_device_id);
CREATE INDEX IF NOT EXISTS idx_results_target ON results(target_device_id);

-- Notification settings (SMTP + ntfy defaults)
CREATE TABLE IF NOT EXISTS notification_settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

-- Alert rules
CREATE TABLE IF NOT EXISTS alert_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    event_type TEXT NOT NULL,  -- 'speed_below', 'ping_above', 'packet_loss_above', 'test_error'
    threshold REAL,            -- NULL for test_error
    source_device_id INTEGER,  -- NULL = global (all pairs)
    target_device_id INTEGER,  -- NULL = global (all pairs)
    notify_ntfy BOOLEAN DEFAULT 0,
    ntfy_topic TEXT,           -- Override default topic per rule
    notify_email BOOLEAN DEFAULT 0,
    email_recipients TEXT,     -- Comma-separated emails
    enabled BOOLEAN DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(source_device_id) REFERENCES devices(id) ON DELETE CASCADE,
    FOREIGN KEY(target_device_id) REFERENCES devices(id) ON DELETE CASCADE
);
