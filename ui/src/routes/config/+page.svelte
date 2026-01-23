<script>
    import { onMount } from 'svelte';
    import { getSchedules, updateSchedule, getDevices, addDevice, deleteDevice, getNotificationSettings, updateNotificationSettings, getAlertRules, createAlertRule, deleteAlertRule } from '$lib/api';

    /** @type {import('$lib/api').Schedule[]} */
    let schedules = [];
    /** @type {import('$lib/api').Device[]} */
    let devices = [];
    /** @type {import('$lib/api').NotificationSettings|null} */
    let notificationSettings = null;
    /** @type {import('$lib/api').AlertRule[]} */
    let alertRules = [];
    let loading = true;
    /** @type {string|null} */
    let error = null;
    let saving = false;
    let savingNotifications = false;

    // Toast notification state
    /** @type {{message: string, type: 'success' | 'error', id: number}[]} */
    let toasts = [];
    let toastId = 0;

    /**
     * Show a toast notification
     * @param {string} message
     * @param {'success' | 'error'} type
     */
    function showToast(message, type = 'success') {
        const id = ++toastId;
        toasts = [...toasts, { message, type, id }];

        // Auto-dismiss after 3s for success, 5s for error
        setTimeout(() => {
            toasts = toasts.filter(t => t.id !== id);
        }, type === 'success' ? 3000 : 5000);
    }

    /**
     * @param {number} id
     */
    function dismissToast(id) {
        toasts = toasts.filter(t => t.id !== id);
    }

    // Form for new device
    let newDevice = {
        name: '',
        hostname: '',
        ip: '',
        ssh_user: 'root',
        ssh_port: 22
    };

    // Form for new alert rule
    let showNewRuleForm = false;
    let newRule = {
        name: '',
        event_type: 'ping_above',
        threshold: 100,
        source_device_id: null,
        target_device_id: null,
        notify_ntfy: false,
        ntfy_topic: '',
        notify_email: false,
        email_recipients: '',
        enabled: true
    };

    const eventTypes = [
        { value: 'speed_below', label: 'Speed Below (Mbps)' },
        { value: 'ping_above', label: 'Ping Above (ms)' },
        { value: 'packet_loss_above', label: 'Packet Loss Above (%)' },
        { value: 'test_error', label: 'Test Error' }
    ];

    async function load() {
        try {
            const [s, d, ns, ar] = await Promise.all([
                getSchedules(),
                getDevices(),
                getNotificationSettings().catch(() => null),
                getAlertRules()
            ]);
            schedules = s || [];
            devices = d || [];
            notificationSettings = ns;
            alertRules = ar || [];
        } catch (e) {
            error = e instanceof Error ? e.message : String(e);
        } finally {
            loading = false;
        }
    }

    /**
     * @param {import('$lib/api').Schedule} schedule
     */
    async function saveSchedule(schedule) {
        saving = true;
        try {
            await updateSchedule(schedule.type, schedule.cron, schedule.enabled);
            showToast('Schedule updated successfully!', 'success');
        } catch (e) {
            showToast('Failed to update: ' + (e instanceof Error ? e.message : String(e)), 'error');
        } finally {
            saving = false;
        }
    }

    async function saveNotificationSettings() {
        if (!notificationSettings) return;
        savingNotifications = true;
        try {
            await updateNotificationSettings(notificationSettings);
            showToast('Notification settings updated!', 'success');
        } catch (e) {
            showToast('Failed to update: ' + (e instanceof Error ? e.message : String(e)), 'error');
        } finally {
            savingNotifications = false;
        }
    }

    async function handleAddDevice() {
        if (!newDevice.name || !newDevice.hostname || !newDevice.ssh_user) {
            showToast('Please fill in Name, Hostname, and SSH User', 'error');
            return;
        }
        try {
            await addDevice(newDevice);
            showToast(`Device "${newDevice.name}" added successfully!`, 'success');
            newDevice = { name: '', hostname: '', ip: '', ssh_user: 'root', ssh_port: 22 };
            await load();
        } catch (e) {
            showToast('Failed to add device: ' + (e instanceof Error ? e.message : String(e)), 'error');
        }
    }

    /**
     * @param {number} id
     */
    async function handleDeleteDevice(id) {
        if (!confirm('Are you sure you want to delete this device?')) return;
        try {
            await deleteDevice(id);
            showToast('Device deleted successfully!', 'success');
            await load();
        } catch (e) {
            showToast('Failed to delete device: ' + (e instanceof Error ? e.message : String(e)), 'error');
        }
    }

    async function handleAddRule() {
        if (!newRule.name) {
            showToast('Please provide a rule name', 'error');
            return;
        }
        try {
            await createAlertRule(newRule);
            showToast('Alert rule created!', 'success');
            newRule = {
                name: '',
                event_type: 'ping_above',
                threshold: 100,
                source_device_id: null,
                target_device_id: null,
                notify_ntfy: false,
                ntfy_topic: '',
                notify_email: false,
                email_recipients: '',
                enabled: true
            };
            showNewRuleForm = false;
            await load();
        } catch (e) {
            showToast('Failed to create rule: ' + (e instanceof Error ? e.message : String(e)), 'error');
        }
    }

    /**
     * @param {number} id
     */
    async function handleDeleteRule(id) {
        if (!confirm('Are you sure you want to delete this alert rule?')) return;
        try {
            await deleteAlertRule(id);
            showToast('Alert rule deleted!', 'success');
            await load();
        } catch (e) {
            showToast('Failed to delete rule: ' + (e instanceof Error ? e.message : String(e)), 'error');
        }
    }

    /**
     * @param {string} eventType
     */
    function getEventTypeLabel(eventType) {
        const type = eventTypes.find(t => t.value === eventType);
        return type ? type.label : eventType;
    }

    /**
     * @param {number|null} deviceId
     */
    function getDeviceName(deviceId) {
        if (deviceId === null) return 'Any';
        const dev = devices.find(d => d.id === deviceId);
        return dev ? dev.name : `Device ${deviceId}`;
    }

    onMount(load);
</script>

<!-- Toast Container -->
<div class="fixed top-4 right-4 z-50 flex flex-col gap-2">
    {#each toasts as toast (toast.id)}
        <div
            class="flex items-center gap-3 px-4 py-3 rounded-lg shadow-lg backdrop-blur-sm animate-slide-in {toast.type === 'success' ? 'bg-green-900/90 border border-green-500/30 text-green-200' : 'bg-red-900/90 border border-red-500/30 text-red-200'}"
            role="alert"
        >
            {#if toast.type === 'success'}
                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-5 h-5 text-green-400 flex-shrink-0">
                    <path stroke-linecap="round" stroke-linejoin="round" d="m4.5 12.75 6 6 9-13.5" />
                </svg>
            {:else}
                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-5 h-5 text-red-400 flex-shrink-0">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9 3.75h.008v.008H12v-.008Z" />
                </svg>
            {/if}
            <span class="text-sm font-medium">{toast.message}</span>
            <button
                onclick={() => dismissToast(toast.id)}
                class="ml-2 text-gray-400 hover:text-white transition-colors"
            >
                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-4 h-4">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12" />
                </svg>
            </button>
        </div>
    {/each}
</div>

<style>
    @keyframes slide-in {
        from {
            transform: translateX(100%);
            opacity: 0;
        }
        to {
            transform: translateX(0);
            opacity: 1;
        }
    }
    .animate-slide-in {
        animation: slide-in 0.3s ease-out;
    }
</style>

<div class="space-y-12">
    <header>
        <h1 class="text-3xl font-bold text-white text-shadow-sm">Configuration</h1>
        <p class="text-gray-400 mt-2">Manage devices, schedules, and notifications.</p>
    </header>

    {#if loading}
        <div class="text-gray-400 animate-pulse">Loading settings...</div>
    {:else if error}
        <div class="p-4 bg-red-900/50 border border-red-800 text-red-200 rounded">Error: {error}</div>
    {:else}
        <!-- Test Schedules Section -->
        <section class="space-y-6">
            <h2 class="text-2xl font-semibold text-white flex items-center gap-2">
                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-6 h-6 text-cyan-400">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" />
                </svg>
                Test Schedules
            </h2>
            <div class="grid gap-6 md:grid-cols-2">
                {#each schedules as schedule}
                    <div class="bg-gray-800/50 border border-gray-700 rounded-xl p-6 backdrop-blur hover:border-gray-600 transition-colors">
                        <div class="flex justify-between items-center mb-6">
                            <h3 class="text-lg font-semibold capitalize text-white">{schedule.type} Test</h3>
                            <span class={`px-2 py-0.5 rounded text-[10px] font-bold tracking-wider ${schedule.enabled ? 'bg-cyan-500/20 text-cyan-400 border border-cyan-500/30' : 'bg-gray-700 text-gray-400'}`}>
                                {schedule.enabled ? 'ACTIVE' : 'INACTIVE'}
                            </span>
                        </div>

                        <div class="space-y-5">
                            <div>
                                <label class="block text-xs font-medium text-gray-400 uppercase tracking-wider mb-2" for="cron-{schedule.id}">Interval (Duration)</label>
                                <input
                                    id="cron-{schedule.id}"
                                    type="text"
                                    bind:value={schedule.cron}
                                    class="w-full bg-gray-900/50 border border-gray-700 rounded-lg px-4 py-2.5 text-white font-mono focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none transition-all"
                                    placeholder="e.g. 1m, 5m, 15m"
                                />
                                <p class="text-[11px] text-gray-500 mt-2">Go duration format: 30s, 1m, 5m, 15m, 1h</p>
                            </div>

                            <div class="flex items-center gap-3">
                                <label class="relative inline-flex items-center cursor-pointer">
                                    <input type="checkbox" bind:checked={schedule.enabled} class="sr-only peer">
                                    <div class="w-11 h-6 bg-gray-700 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-cyan-800 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-cyan-600"></div>
                                    <span class="ms-3 text-sm font-medium text-gray-300">Enabled</span>
                                </label>
                            </div>

                            <button
                                onclick={() => saveSchedule(schedule)}
                                disabled={saving}
                                class="w-full bg-cyan-600 hover:bg-cyan-500 text-white font-semibold py-2.5 rounded-lg transition-all shadow-lg shadow-cyan-900/20 disabled:opacity-50"
                            >
                                {saving ? 'Applying...' : 'Update Schedule'}
                            </button>
                        </div>
                    </div>
                {/each}
            </div>
        </section>

        <!-- Devices Section -->
        <section class="space-y-6">
            <h2 class="text-2xl font-semibold text-white flex items-center gap-2">
                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-6 h-6 text-cyan-400">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M9 17.25v1.007a3 3 0 0 1-.879 2.122L7.5 21h9l-.621-.621A3 3 0 0 1 15 18.257V17.25m6-12V15a2.25 2.25 0 0 1-2.25 2.25H5.25A2.25 2.25 0 0 1 3 15V5.25m18 0A2.25 2.25 0 0 0 18.75 3H5.25A2.25 2.25 0 0 0 3 5.25m18 0V12a2.25 2.25 0 0 1-2.25 2.25H5.25A2.25 2.25 0 0 1 3 12V5.25" />
                </svg>
                Network Devices
            </h2>

            <div class="bg-gray-800/50 border border-gray-700 rounded-xl overflow-hidden backdrop-blur">
                <table class="w-full text-left">
                    <thead class="bg-gray-900/50 text-gray-400 text-xs uppercase tracking-wider font-bold">
                        <tr>
                            <th class="px-6 py-4">Name</th>
                            <th class="px-6 py-4">Hostname / IP</th>
                            <th class="px-6 py-4">SSH Config</th>
                            <th class="px-6 py-4 text-right">Actions</th>
                        </tr>
                    </thead>
                    <tbody class="divide-y divide-gray-700">
                        {#each devices as dev}
                            <tr class="hover:bg-gray-700/30 transition-colors">
                                <td class="px-6 py-4 font-semibold text-white">{dev.name}</td>
                                <td class="px-6 py-4 font-mono text-sm text-gray-300">
                                    {dev.hostname}
                                    {#if dev.ip && dev.ip !== dev.hostname}
                                        <span class="text-gray-500 ml-2">({dev.ip})</span>
                                    {/if}
                                </td>
                                <td class="px-6 py-4 text-xs text-gray-400">
                                    <span class="bg-gray-900 px-2 py-1 rounded">{dev.ssh_user}@{dev.ssh_port}</span>
                                </td>
                                <td class="px-6 py-4 text-right">
                                    <button
                                        onclick={() => handleDeleteDevice(dev.id)}
                                        class="text-gray-500 hover:text-red-400 transition-colors p-1"
                                        title="Delete Device"
                                    >
                                        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-5 h-5">
                                            <path stroke-linecap="round" stroke-linejoin="round" d="m14.74 9-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 0 1-2.244 2.077H8.084a2.25 2.25 0 0 1-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 0 0-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 0 1 3.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 0 0-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 0 0-7.5 0" />
                                        </svg>
                                    </button>
                                </td>
                            </tr>
                        {/each}
                        <!-- Add New Device Row -->
                        <tr class="bg-cyan-900/5">
                            <td class="px-6 py-4">
                                <input type="text" bind:value={newDevice.name} placeholder="Device Name" class="bg-gray-900 border border-gray-700 rounded px-3 py-1.5 text-sm w-full outline-none focus:border-cyan-500"/>
                            </td>
                            <td class="px-6 py-4 flex gap-2">
                                <input type="text" bind:value={newDevice.hostname} placeholder="Hostname" class="bg-gray-900 border border-gray-700 rounded px-3 py-1.5 text-sm w-full outline-none focus:border-cyan-500"/>
                                <input type="text" bind:value={newDevice.ip} placeholder="IP (Optional)" class="bg-gray-900 border border-gray-700 rounded px-3 py-1.5 text-sm w-full outline-none focus:border-cyan-500"/>
                            </td>
                            <td class="px-6 py-4">
                                <div class="flex gap-2">
                                    <input type="text" bind:value={newDevice.ssh_user} placeholder="User" class="bg-gray-900 border border-gray-700 rounded px-3 py-1.5 text-sm w-20 outline-none focus:border-cyan-500"/>
                                    <input type="number" bind:value={newDevice.ssh_port} placeholder="Port" class="bg-gray-900 border border-gray-700 rounded px-3 py-1.5 text-sm w-16 outline-none focus:border-cyan-500"/>
                                </div>
                            </td>
                            <td class="px-6 py-4 text-right">
                                <button
                                    onclick={handleAddDevice}
                                    class="bg-cyan-600 hover:bg-cyan-500 text-white p-2 rounded-lg transition-colors shadow-lg"
                                    title="Add Device"
                                >
                                    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-5 h-5">
                                        <path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
                                    </svg>
                                </button>
                            </td>
                        </tr>
                    </tbody>
                </table>
            </div>
        </section>

        <!-- Notification Settings Section -->
        {#if notificationSettings}
            <section class="space-y-6">
                <h2 class="text-2xl font-semibold text-white flex items-center gap-2">
                    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-6 h-6 text-cyan-400">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M14.857 17.082a23.848 23.848 0 0 0 5.454-1.31A8.967 8.967 0 0 1 18 9.75V9A6 6 0 0 0 6 9v.75a8.967 8.967 0 0 1-2.312 6.022c1.733.64 3.56 1.085 5.455 1.31m5.714 0a24.255 24.255 0 0 1-5.714 0m5.714 0a3 3 0 1 1-5.714 0" />
                    </svg>
                    Notification Settings
                </h2>

                <div class="grid gap-6 md:grid-cols-2">
                    <!-- ntfy Settings -->
                    <div class="bg-gray-800/50 border border-gray-700 rounded-xl p-6 backdrop-blur">
                        <div class="flex justify-between items-center mb-6">
                            <h3 class="text-lg font-semibold text-white">ntfy Push Notifications</h3>
                            {#if notificationSettings.env_configured.ntfy_enabled || notificationSettings.env_configured.ntfy_server || notificationSettings.env_configured.ntfy_topic}
                                <span class="px-2 py-0.5 rounded text-[10px] font-bold tracking-wider bg-yellow-500/20 text-yellow-400 border border-yellow-500/30">
                                    ENV CONFIGURED
                                </span>
                            {/if}
                        </div>

                        <div class="space-y-4">
                            <div class="flex items-center gap-3">
                                <label class="relative inline-flex items-center cursor-pointer">
                                    <input
                                        type="checkbox"
                                        bind:checked={notificationSettings.ntfy.enabled}
                                        disabled={notificationSettings.env_configured.ntfy_enabled}
                                        class="sr-only peer"
                                    >
                                    <div class="w-11 h-6 bg-gray-700 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-cyan-800 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-cyan-600 peer-disabled:opacity-50"></div>
                                    <span class="ms-3 text-sm font-medium text-gray-300">Enabled</span>
                                </label>
                            </div>

                            <div>
                                <label class="block text-xs font-medium text-gray-400 uppercase tracking-wider mb-2">Server URL</label>
                                <input
                                    type="text"
                                    bind:value={notificationSettings.ntfy.server}
                                    disabled={notificationSettings.env_configured.ntfy_server}
                                    class="w-full bg-gray-900/50 border border-gray-700 rounded-lg px-4 py-2.5 text-white font-mono text-sm focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none transition-all disabled:opacity-50"
                                    placeholder="https://ntfy.sh"
                                />
                            </div>

                            <div>
                                <label class="block text-xs font-medium text-gray-400 uppercase tracking-wider mb-2">Default Topic</label>
                                <input
                                    type="text"
                                    bind:value={notificationSettings.ntfy.topic}
                                    disabled={notificationSettings.env_configured.ntfy_topic}
                                    class="w-full bg-gray-900/50 border border-gray-700 rounded-lg px-4 py-2.5 text-white font-mono text-sm focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none transition-all disabled:opacity-50"
                                    placeholder="homelab-alerts"
                                />
                            </div>

                            <div>
                                <label class="block text-xs font-medium text-gray-400 uppercase tracking-wider mb-2">Access Token (Optional)</label>
                                <input
                                    type="password"
                                    bind:value={notificationSettings.ntfy.token}
                                    disabled={notificationSettings.env_configured.ntfy_token}
                                    class="w-full bg-gray-900/50 border border-gray-700 rounded-lg px-4 py-2.5 text-white font-mono text-sm focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none transition-all disabled:opacity-50"
                                    placeholder="tk_xxx..."
                                />
                            </div>
                        </div>
                    </div>

                    <!-- SMTP Settings -->
                    <div class="bg-gray-800/50 border border-gray-700 rounded-xl p-6 backdrop-blur">
                        <div class="flex justify-between items-center mb-6">
                            <h3 class="text-lg font-semibold text-white">Email (SMTP)</h3>
                            {#if notificationSettings.env_configured.smtp_enabled || notificationSettings.env_configured.smtp_host}
                                <span class="px-2 py-0.5 rounded text-[10px] font-bold tracking-wider bg-yellow-500/20 text-yellow-400 border border-yellow-500/30">
                                    ENV CONFIGURED
                                </span>
                            {/if}
                        </div>

                        <div class="space-y-4">
                            <div class="flex items-center gap-3">
                                <label class="relative inline-flex items-center cursor-pointer">
                                    <input
                                        type="checkbox"
                                        bind:checked={notificationSettings.smtp.enabled}
                                        disabled={notificationSettings.env_configured.smtp_enabled}
                                        class="sr-only peer"
                                    >
                                    <div class="w-11 h-6 bg-gray-700 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-cyan-800 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-cyan-600 peer-disabled:opacity-50"></div>
                                    <span class="ms-3 text-sm font-medium text-gray-300">Enabled</span>
                                </label>
                            </div>

                            <div class="grid grid-cols-3 gap-3">
                                <div class="col-span-2">
                                    <label class="block text-xs font-medium text-gray-400 uppercase tracking-wider mb-2">SMTP Host</label>
                                    <input
                                        type="text"
                                        bind:value={notificationSettings.smtp.host}
                                        disabled={notificationSettings.env_configured.smtp_host}
                                        class="w-full bg-gray-900/50 border border-gray-700 rounded-lg px-4 py-2.5 text-white font-mono text-sm focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none transition-all disabled:opacity-50"
                                        placeholder="smtp.gmail.com"
                                    />
                                </div>
                                <div>
                                    <label class="block text-xs font-medium text-gray-400 uppercase tracking-wider mb-2">Port</label>
                                    <input
                                        type="number"
                                        bind:value={notificationSettings.smtp.port}
                                        disabled={notificationSettings.env_configured.smtp_port}
                                        class="w-full bg-gray-900/50 border border-gray-700 rounded-lg px-4 py-2.5 text-white font-mono text-sm focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none transition-all disabled:opacity-50"
                                        placeholder="587"
                                    />
                                </div>
                            </div>

                            <div>
                                <label class="block text-xs font-medium text-gray-400 uppercase tracking-wider mb-2">Username</label>
                                <input
                                    type="text"
                                    bind:value={notificationSettings.smtp.user}
                                    disabled={notificationSettings.env_configured.smtp_user}
                                    class="w-full bg-gray-900/50 border border-gray-700 rounded-lg px-4 py-2.5 text-white font-mono text-sm focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none transition-all disabled:opacity-50"
                                    placeholder="user@gmail.com"
                                />
                            </div>

                            <div>
                                <label class="block text-xs font-medium text-gray-400 uppercase tracking-wider mb-2">Password</label>
                                <input
                                    type="password"
                                    bind:value={notificationSettings.smtp.password}
                                    disabled={notificationSettings.env_configured.smtp_password}
                                    class="w-full bg-gray-900/50 border border-gray-700 rounded-lg px-4 py-2.5 text-white font-mono text-sm focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none transition-all disabled:opacity-50"
                                    placeholder="app-password"
                                />
                            </div>

                            <div>
                                <label class="block text-xs font-medium text-gray-400 uppercase tracking-wider mb-2">From Address</label>
                                <input
                                    type="text"
                                    bind:value={notificationSettings.smtp.from}
                                    disabled={notificationSettings.env_configured.smtp_from}
                                    class="w-full bg-gray-900/50 border border-gray-700 rounded-lg px-4 py-2.5 text-white font-mono text-sm focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none transition-all disabled:opacity-50"
                                    placeholder="homelab@example.com"
                                />
                            </div>
                        </div>
                    </div>
                </div>

                <button
                    onclick={saveNotificationSettings}
                    disabled={savingNotifications}
                    class="bg-cyan-600 hover:bg-cyan-500 text-white font-semibold py-2.5 px-6 rounded-lg transition-all shadow-lg shadow-cyan-900/20 disabled:opacity-50"
                >
                    {savingNotifications ? 'Saving...' : 'Save Notification Settings'}
                </button>
            </section>
        {/if}

        <!-- Alert Rules Section -->
        <section class="space-y-6">
            <div class="flex items-center justify-between">
                <h2 class="text-2xl font-semibold text-white flex items-center gap-2">
                    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-6 h-6 text-cyan-400">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126ZM12 15.75h.007v.008H12v-.008Z" />
                    </svg>
                    Alert Rules
                </h2>
                <button
                    onclick={() => showNewRuleForm = !showNewRuleForm}
                    class="flex items-center gap-2 px-4 py-2 bg-cyan-600 hover:bg-cyan-500 text-white rounded-lg transition-colors text-sm font-semibold"
                >
                    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-4 h-4">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
                    </svg>
                    Add Rule
                </button>
            </div>

            <!-- New Rule Form -->
            {#if showNewRuleForm}
                <div class="bg-gray-800/50 border border-cyan-500/30 rounded-xl p-6 backdrop-blur">
                    <h3 class="text-lg font-semibold text-white mb-4">New Alert Rule</h3>
                    <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
                        <div>
                            <label class="block text-xs font-medium text-gray-400 uppercase tracking-wider mb-2">Rule Name</label>
                            <input
                                type="text"
                                bind:value={newRule.name}
                                class="w-full bg-gray-900/50 border border-gray-700 rounded-lg px-4 py-2.5 text-white text-sm focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none transition-all"
                                placeholder="High Latency Alert"
                            />
                        </div>

                        <div>
                            <label class="block text-xs font-medium text-gray-400 uppercase tracking-wider mb-2">Event Type</label>
                            <select
                                bind:value={newRule.event_type}
                                class="w-full bg-gray-900/50 border border-gray-700 rounded-lg px-4 py-2.5 text-white text-sm focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none transition-all"
                            >
                                {#each eventTypes as et}
                                    <option value={et.value}>{et.label}</option>
                                {/each}
                            </select>
                        </div>

                        {#if newRule.event_type !== 'test_error'}
                            <div>
                                <label class="block text-xs font-medium text-gray-400 uppercase tracking-wider mb-2">Threshold</label>
                                <input
                                    type="number"
                                    bind:value={newRule.threshold}
                                    step="0.1"
                                    class="w-full bg-gray-900/50 border border-gray-700 rounded-lg px-4 py-2.5 text-white text-sm focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none transition-all"
                                />
                            </div>
                        {/if}

                        <div>
                            <label class="block text-xs font-medium text-gray-400 uppercase tracking-wider mb-2">Source Device</label>
                            <select
                                bind:value={newRule.source_device_id}
                                class="w-full bg-gray-900/50 border border-gray-700 rounded-lg px-4 py-2.5 text-white text-sm focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none transition-all"
                            >
                                <option value={null}>Any (Global)</option>
                                {#each devices as dev}
                                    <option value={dev.id}>{dev.name}</option>
                                {/each}
                            </select>
                        </div>

                        <div>
                            <label class="block text-xs font-medium text-gray-400 uppercase tracking-wider mb-2">Target Device</label>
                            <select
                                bind:value={newRule.target_device_id}
                                class="w-full bg-gray-900/50 border border-gray-700 rounded-lg px-4 py-2.5 text-white text-sm focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none transition-all"
                            >
                                <option value={null}>Any (Global)</option>
                                {#each devices as dev}
                                    <option value={dev.id}>{dev.name}</option>
                                {/each}
                            </select>
                        </div>
                    </div>

                    <div class="grid gap-4 md:grid-cols-2 mt-4">
                        <div class="space-y-3">
                            <div class="flex items-center gap-3">
                                <label class="relative inline-flex items-center cursor-pointer">
                                    <input type="checkbox" bind:checked={newRule.notify_ntfy} class="sr-only peer">
                                    <div class="w-11 h-6 bg-gray-700 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-cyan-800 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-cyan-600"></div>
                                    <span class="ms-3 text-sm font-medium text-gray-300">Notify via ntfy</span>
                                </label>
                            </div>
                            {#if newRule.notify_ntfy}
                                <input
                                    type="text"
                                    bind:value={newRule.ntfy_topic}
                                    class="w-full bg-gray-900/50 border border-gray-700 rounded-lg px-4 py-2.5 text-white text-sm focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none transition-all"
                                    placeholder="Topic override (optional)"
                                />
                            {/if}
                        </div>

                        <div class="space-y-3">
                            <div class="flex items-center gap-3">
                                <label class="relative inline-flex items-center cursor-pointer">
                                    <input type="checkbox" bind:checked={newRule.notify_email} class="sr-only peer">
                                    <div class="w-11 h-6 bg-gray-700 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-cyan-800 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-cyan-600"></div>
                                    <span class="ms-3 text-sm font-medium text-gray-300">Notify via Email</span>
                                </label>
                            </div>
                            {#if newRule.notify_email}
                                <input
                                    type="text"
                                    bind:value={newRule.email_recipients}
                                    class="w-full bg-gray-900/50 border border-gray-700 rounded-lg px-4 py-2.5 text-white text-sm focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none transition-all"
                                    placeholder="email1@example.com, email2@example.com"
                                />
                            {/if}
                        </div>
                    </div>

                    <div class="flex gap-3 mt-6">
                        <button
                            onclick={handleAddRule}
                            class="bg-cyan-600 hover:bg-cyan-500 text-white font-semibold py-2.5 px-6 rounded-lg transition-all"
                        >
                            Create Rule
                        </button>
                        <button
                            onclick={() => showNewRuleForm = false}
                            class="bg-gray-700 hover:bg-gray-600 text-white font-semibold py-2.5 px-6 rounded-lg transition-all"
                        >
                            Cancel
                        </button>
                    </div>
                </div>
            {/if}

            <!-- Rules Table -->
            <div class="bg-gray-800/50 border border-gray-700 rounded-xl overflow-hidden backdrop-blur">
                <table class="w-full text-left">
                    <thead class="bg-gray-900/50 text-gray-400 text-xs uppercase tracking-wider font-bold">
                        <tr>
                            <th class="px-6 py-4">Name</th>
                            <th class="px-6 py-4">Condition</th>
                            <th class="px-6 py-4">Scope</th>
                            <th class="px-6 py-4">Notify</th>
                            <th class="px-6 py-4">Status</th>
                            <th class="px-6 py-4 text-right">Actions</th>
                        </tr>
                    </thead>
                    <tbody class="divide-y divide-gray-700">
                        {#each alertRules as rule}
                            <tr class="hover:bg-gray-700/30 transition-colors">
                                <td class="px-6 py-4 font-semibold text-white">{rule.name}</td>
                                <td class="px-6 py-4 text-sm text-gray-300">
                                    {getEventTypeLabel(rule.event_type)}
                                    {#if rule.threshold !== null}
                                        <span class="text-cyan-400 font-mono ml-1">{rule.threshold}</span>
                                    {/if}
                                </td>
                                <td class="px-6 py-4 text-sm text-gray-400">
                                    {getDeviceName(rule.source_device_id)} &rarr; {getDeviceName(rule.target_device_id)}
                                </td>
                                <td class="px-6 py-4">
                                    <div class="flex gap-2">
                                        {#if rule.notify_ntfy}
                                            <span class="px-2 py-0.5 rounded text-[10px] font-bold bg-purple-500/20 text-purple-400 border border-purple-500/30">NTFY</span>
                                        {/if}
                                        {#if rule.notify_email}
                                            <span class="px-2 py-0.5 rounded text-[10px] font-bold bg-blue-500/20 text-blue-400 border border-blue-500/30">EMAIL</span>
                                        {/if}
                                    </div>
                                </td>
                                <td class="px-6 py-4">
                                    <span class={`px-2 py-0.5 rounded text-[10px] font-bold tracking-wider ${rule.enabled ? 'bg-green-500/20 text-green-400 border border-green-500/30' : 'bg-gray-700 text-gray-400'}`}>
                                        {rule.enabled ? 'ACTIVE' : 'DISABLED'}
                                    </span>
                                </td>
                                <td class="px-6 py-4 text-right">
                                    <button
                                        onclick={() => handleDeleteRule(rule.id)}
                                        class="text-gray-500 hover:text-red-400 transition-colors p-1"
                                        title="Delete Rule"
                                    >
                                        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-5 h-5">
                                            <path stroke-linecap="round" stroke-linejoin="round" d="m14.74 9-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 0 1-2.244 2.077H8.084a2.25 2.25 0 0 1-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 0 0-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 0 1 3.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 0 0-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 0 0-7.5 0" />
                                        </svg>
                                    </button>
                                </td>
                            </tr>
                        {:else}
                            <tr>
                                <td colspan="6" class="px-6 py-8 text-center text-gray-500">
                                    No alert rules configured. Click "Add Rule" to create one.
                                </td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
            </div>
        </section>
    {/if}
</div>
