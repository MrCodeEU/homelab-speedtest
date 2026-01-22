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
 * @returns {Promise<Result[]>}
 */
export async function getHistory(limit = 100) {
    const res = await fetch(`${API_BASE}/history?limit=${limit}`);
    if (!res.ok) throw new Error('Failed to fetch history');
    return res.json();
}
