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
    type TEXT NOT NULL, -- 'ping', 'speed'
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
