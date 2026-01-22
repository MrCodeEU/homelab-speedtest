<script>
    import { onMount } from 'svelte';
    import { getSchedules, updateSchedule, getDevices, addDevice, deleteDevice } from '$lib/api';

    /** @type {import('$lib/api').Schedule[]} */
    let schedules = [];
    /** @type {import('$lib/api').Device[]} */
    let devices = [];
    let loading = true;
    /** @type {string|null} */
    let error = null;
    let saving = false;

    // Form for new device
    let newDevice = {
        name: '',
        hostname: '',
        ip: '',
        ssh_user: 'root',
        ssh_port: 22
    };

    async function load() {
        try {
            const [s, d] = await Promise.all([getSchedules(), getDevices()]);
            schedules = s || [];
            devices = d || [];
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
            alert('Schedule updated!');
        } catch (e) {
            alert('Failed to update: ' + (e instanceof Error ? e.message : String(e)));
        } finally {
            saving = false;
        }
    }

    async function handleAddDevice() {
        if (!newDevice.name || !newDevice.hostname || !newDevice.ssh_user) {
            alert('Please fill in Name, Hostname, and SSH User');
            return;
        }
        try {
            await addDevice(newDevice);
            newDevice = { name: '', hostname: '', ip: '', ssh_user: 'root', ssh_port: 22 };
            await load();
        } catch (e) {
            alert('Failed to add device: ' + (e instanceof Error ? e.message : String(e)));
        }
    }

    /**
     * @param {number} id
     */
    async function handleDeleteDevice(id) {
        if (!confirm('Are you sure you want to delete this device?')) return;
        try {
            await deleteDevice(id);
            await load();
        } catch (e) {
            alert('Failed to delete device: ' + (e instanceof Error ? e.message : String(e)));
        }
    }

    onMount(load);
</script>

<div class="space-y-12">
    <header>
        <h1 class="text-3xl font-bold text-white text-shadow-sm">Configuration</h1>
        <p class="text-gray-400 mt-2">Manage devices and test schedules.</p>
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
                                <label class="block text-xs font-medium text-gray-400 uppercase tracking-wider mb-2" for="cron-{schedule.id}">Frequency</label>
                                <input 
                                    id="cron-{schedule.id}"
                                    type="text" 
                                    bind:value={schedule.cron} 
                                    class="w-full bg-gray-900/50 border border-gray-700 rounded-lg px-4 py-2.5 text-white font-mono focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none transition-all"
                                    placeholder="e.g. 1m, 30s"
                                />
                                <p class="text-[11px] text-gray-500 mt-2">Go duration format (e.g. '30s', '5m', '1h')</p>
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
    {/if}
</div>