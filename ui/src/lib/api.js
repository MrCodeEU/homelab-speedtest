/**
 * @typedef {Object} Device
 * @property {number} id
 * @property {string} name
 * @property {string} hostname
 * @property {string} ip
 * @property {string} ssh_user
 * @property {number} ssh_port
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
