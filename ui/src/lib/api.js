/**
 * @typedef {Object} Device
 * @property {number} id
 * @property {string} name
 * @property {string} hostname
 * @property {string} ip
 * @property {string} ssh_user
 * @property {number} ssh_port
 */

/**
 * @typedef {Object} Result
 * @property {number} source_id
 * @property {number} target_id
 * @property {string} type
 * @property {number} latency_ms
 * @property {number} bandwidth_mbps
 * @property {string} timestamp
 * @property {string} error
 */

const API_BASE = '/api';

/**
 * Fetch all devices
 * @returns {Promise<Device[]>}
 */
export async function getDevices() {
    const res = await fetch(`${API_BASE}/devices`);
    if (!res.ok) throw new Error('Failed to fetch devices');
    return res.json();
}

/**
 * Fetch latest results
 * @returns {Promise<Result[]>}
 */
export async function getResults() {
    const res = await fetch(`${API_BASE}/results/latest`);
    if (!res.ok) throw new Error('Failed to fetch results');
    return res.json();
}

/**
 * Add a new device
 * @param {Omit<Device, 'id'>} device
 */
export async function addDevice(device) {
    const res = await fetch(`${API_BASE}/devices`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(device),
    });
    if (!res.ok) throw new Error('Failed to add device');
}

/**
 * Update a device
 * @param {Device} device
 */
export async function updateDevice(device) {
    const res = await fetch(`${API_BASE}/devices/${device.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(device),
    });
    if (!res.ok) throw new Error('Failed to update device');
}

/**
 * Delete a device
 * @param {number} id
 */
export async function deleteDevice(id) {
    const res = await fetch(`${API_BASE}/devices/${id}`, {
        method: 'DELETE',
    });
    if (!res.ok) throw new Error('Failed to delete device');
}

/**
 * @typedef {Object} Schedule
 * @property {number} id
 * @property {string} type
 * @property {string} cron
 * @property {boolean} enabled
 */

/**
 * Fetch schedules
 * @returns {Promise<Schedule[]>}
 */
export async function getSchedules() {
    const res = await fetch(`${API_BASE}/schedules`);
    if (!res.ok) throw new Error('Failed to fetch schedules');
    return res.json();
}

/**
 * Update a schedule
 * @param {string} type
 * @param {string} cron
 * @param {boolean} enabled
 */
export async function updateSchedule(type, cron, enabled) {
    const res = await fetch(`${API_BASE}/schedules`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ type, cron, enabled }),
    });
    if (!res.ok) throw new Error('Failed to update schedule');
}

/**
 * Fetch history
 * @param {number} [limit]
 * @param {string} [type]
 * @returns {Promise<Result[]>}
 */
export async function getHistory(limit = 100, type = '') {
    const url = new URL(`${window.location.origin}${API_BASE}/history`);
    url.searchParams.append('limit', limit.toString());
    if (type) {
        url.searchParams.append('type', type);
    }
    const res = await fetch(url.toString());
    if (!res.ok) throw new Error('Failed to fetch history');
    return res.json();
}

/**
 * Trigger all pings manually
 */
export async function triggerPingAll() {
    const res = await fetch(`${API_BASE}/test/ping/all`, { method: 'POST' });
    if (!res.ok) throw new Error('Failed to trigger pings');
}

/**
 * Trigger all speed tests manually
 */
export async function triggerSpeedAll() {
    const res = await fetch(`${API_BASE}/test/speed/all`, { method: 'POST' });
    if (!res.ok) throw new Error('Failed to trigger speed tests');
}

/**
 * @typedef {Object} ScheduleStatus
 * @property {string} type
 * @property {string} interval
 * @property {boolean} enabled
 * @property {string} next_run
 */

/**
 * Fetch schedule status with next run times
 * @returns {Promise<ScheduleStatus[]>}
 */
export async function getScheduleStatus() {
    const res = await fetch(`${API_BASE}/schedule-status`);
    if (!res.ok) throw new Error('Failed to fetch schedule status');
    return res.json();
}

// Queue Status

/**
 * @typedef {Object} Task
 * @property {string} id
 * @property {string} type
 * @property {number} priority
 * @property {string} created_at
 */

/**
 * @typedef {Object} QueueStatus
 * @property {Task|null} running
 * @property {Task[]} queued
 * @property {number} length
 */

/**
 * Fetch queue status
 * @returns {Promise<QueueStatus>}
 */
export async function getQueueStatus() {
    const res = await fetch(`${API_BASE}/queue-status`);
    if (!res.ok) throw new Error('Failed to fetch queue status');
    return res.json();
}

// Notification Settings

/**
 * @typedef {Object} NtfySettings
 * @property {boolean} enabled
 * @property {string} server
 * @property {string} topic
 * @property {string} token
 */

/**
 * @typedef {Object} SMTPSettings
 * @property {boolean} enabled
 * @property {string} host
 * @property {number} port
 * @property {string} user
 * @property {string} password
 * @property {string} from
 * @property {boolean} skip_ssl_verify
 */

/**
 * @typedef {Object} EnvConfigStatus
 * @property {boolean} ntfy_enabled
 * @property {boolean} ntfy_server
 * @property {boolean} ntfy_topic
 * @property {boolean} ntfy_token
 * @property {boolean} smtp_enabled
 * @property {boolean} smtp_host
 * @property {boolean} smtp_port
 * @property {boolean} smtp_user
 * @property {boolean} smtp_password
 * @property {boolean} smtp_from
 */

/**
 * @typedef {Object} NotificationSettings
 * @property {NtfySettings} ntfy
 * @property {SMTPSettings} smtp
 * @property {EnvConfigStatus} env_configured
 */

/**
 * Fetch notification settings
 * @returns {Promise<NotificationSettings>}
 */
export async function getNotificationSettings() {
    const res = await fetch(`${API_BASE}/notification-settings`);
    if (!res.ok) throw new Error('Failed to fetch notification settings');
    return res.json();
}

/**
 * Update notification settings
 * @param {NotificationSettings} settings
 */
export async function updateNotificationSettings(settings) {
    const res = await fetch(`${API_BASE}/notification-settings`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(settings),
    });
    if (!res.ok) throw new Error('Failed to update notification settings');
}

/**
 * Test ntfy notification
 * @param {NtfySettings} [settings]
 */
export async function testNtfy(settings) {
    const res = await fetch(`${API_BASE}/notify/test/ntfy`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: settings ? JSON.stringify(settings) : undefined,
    });
    if (!res.ok) {
        const text = await res.text();
        throw new Error(text || 'Failed to send test ntfy notification');
    }
}

/**
 * Test email notification
 * @param {string} recipients
 * @param {SMTPSettings} [settings]
 */
export async function testEmail(recipients, settings) {
    const res = await fetch(`${API_BASE}/notify/test/email`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ recipients, settings }),
    });
    if (!res.ok) {
        const text = await res.text();
        throw new Error(text || 'Failed to send test email');
    }
}

// Alert Rules

/**
 * @typedef {Object} AlertRule
 * @property {number} id
 * @property {string} name
 * @property {string} event_type
 * @property {number|null} threshold
 * @property {number|null} source_device_id
 * @property {number|null} target_device_id
 * @property {boolean} notify_ntfy
 * @property {string} ntfy_topic
 * @property {boolean} notify_email
 * @property {string} email_recipients
 * @property {boolean} enabled
 * @property {string} created_at
 */

/**
 * Fetch all alert rules
 * @returns {Promise<AlertRule[]>}
 */
export async function getAlertRules() {
    const res = await fetch(`${API_BASE}/alert-rules`);
    if (!res.ok) throw new Error('Failed to fetch alert rules');
    return res.json();
}

/**
 * Create a new alert rule
 * @param {Omit<AlertRule, 'id' | 'created_at'>} rule
 * @returns {Promise<{id: number}>}
 */
export async function createAlertRule(rule) {
    const res = await fetch(`${API_BASE}/alert-rules`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(rule),
    });
    if (!res.ok) throw new Error('Failed to create alert rule');
    return res.json();
}

/**
 * Update an alert rule
 * @param {AlertRule} rule
 */
export async function updateAlertRule(rule) {
    const res = await fetch(`${API_BASE}/alert-rules/${rule.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(rule),
    });
    if (!res.ok) throw new Error('Failed to update alert rule');
}

/**
 * Delete an alert rule
 * @param {number} id
 */
export async function deleteAlertRule(id) {
    const res = await fetch(`${API_BASE}/alert-rules/${id}`, {
        method: 'DELETE',
    });
    if (!res.ok) throw new Error('Failed to delete alert rule');
}

