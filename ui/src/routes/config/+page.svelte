<script>
    import { onMount } from 'svelte';
    import { getDevices, addDevice, deleteDevice } from '$lib/api';

    /** @type {import('$lib/api').Device[]} */
    let devices = [];
    let loading = true;
    /** @type {string|null} */
    let error = null;

    let newDevice = {
        name: '',
        hostname: '',
        ip: '',
        ssh_user: 'root',
        ssh_port: 22
    };

    async function load() {
        loading = true;
        try {
            devices = await getDevices();
        } catch (/** @type {any} */ e) {
            error = e.message;
        } finally {
            loading = false;
        }
    }

    onMount(load);

    async function handleSubmit() {
        try {
            await addDevice(newDevice);
            // Reset form
            newDevice = { name: '', hostname: '', ip: '', ssh_user: 'root', ssh_port: 22 };
            await load();
        } catch (/** @type {any} */ e) {
            alert('Failed to add device: ' + e.message);
        }
    }

    /** @param {number} id */
    async function handleDelete(id) {
        if (!confirm('Are you sure you want to delete this device?')) return;
        try {
            await deleteDevice(id);
            await load();
        } catch (/** @type {any} */ e) {
            alert('Failed to delete device: ' + e.message);
        }
    }
</script>

<div class="max-w-4xl mx-auto space-y-12">
    <!-- Managed Devices -->
    <section class="bg-gray-800/50 border border-gray-700 rounded-xl p-8 backdrop-blur">
        <h2 class="text-2xl font-bold mb-6">Managed Devices</h2>
        
        {#if loading}
            <div class="text-gray-400 animate-pulse">Loading devices...</div>
        {:else if error}
             <div class="p-4 bg-red-900/30 border border-red-800 text-red-200 rounded-lg">
                Error loading devices: {error}
             </div>
        {:else if devices.length === 0}
            <div class="text-gray-500 italic text-center py-8 bg-gray-900/30 rounded-lg border border-gray-800 border-dashed">
                No devices configured yet. Add one below!
            </div>
        {:else}
            <div class="overflow-x-auto">
                <table class="w-full text-left border-collapse">
                    <thead>
                        <tr class="text-gray-400 border-b border-gray-700">
                            <th class="py-3 px-4">Name</th>
                            <th class="py-3 px-4">Hostname / IP</th>
                            <th class="py-3 px-4">SSH</th>
                            <th class="py-3 px-4 text-right">Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        {#each devices as device (device.id)}
                            <tr class="border-b border-gray-700/50 hover:bg-gray-700/30 transition-colors group">
                                <td class="py-3 px-4 font-medium text-cyan-400">{device.name}</td>
                                <td class="py-3 px-4 text-gray-300">
                                    <div class="flex flex-col">
                                        <span>{device.hostname}</span>
                                        {#if device.ip}<span class="text-xs text-gray-500 font-mono">{device.ip}</span>{/if}
                                    </div>
                                </td>
                                <td class="py-3 px-4 text-gray-400 text-sm font-mono">
                                    {device.ssh_user}@{device.hostname}:{device.ssh_port}
                                </td>
                                <td class="py-3 px-4 text-right">
                                    <button on:click={() => handleDelete(device.id)} 
                                            class="text-gray-500 hover:text-red-400 hover:bg-red-900/20 px-3 py-1.5 rounded transition-all opacity-75 group-hover:opacity-100 flex items-center gap-2 ml-auto">
                                        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-4 h-4">
                                          <path stroke-linecap="round" stroke-linejoin="round" d="m14.74 9-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 0 1-2.244 2.077H8.084a2.25 2.25 0 0 1-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 0 0-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 0 1 3.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 0 0-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 0 0-7.5 0" />
                                        </svg>
                                        Remove
                                    </button>
                                </td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
            </div>
        {/if}
    </section>

    <!-- Add Device -->
    <section class="bg-gray-800/50 border border-gray-700 rounded-xl p-8 backdrop-blur">
        <h2 class="text-2xl font-bold mb-6">Add New Device</h2>
        <form on:submit|preventDefault={handleSubmit} class="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div class="space-y-2">
                <label for="name" class="block text-sm font-medium text-gray-400">Device Name</label>
                <input id="name" type="text" bind:value={newDevice.name} required
                       class="w-full bg-gray-900 border border-gray-700 rounded px-4 py-2 focus:ring-2 focus:ring-cyan-500 focus:border-transparent outline-none transition-all"
                       placeholder="e.g. NAS" />
            </div>

            <div class="space-y-2">
                <label for="hostname" class="block text-sm font-medium text-gray-400">Hostname / FQDN</label>
                <input id="hostname" type="text" bind:value={newDevice.hostname} required
                       class="w-full bg-gray-900 border border-gray-700 rounded px-4 py-2 focus:ring-2 focus:ring-cyan-500 focus:border-transparent outline-none transition-all"
                       placeholder="e.g. nas.tail823.ts.net" />
            </div>

            <div class="space-y-2">
                <label for="ip" class="block text-sm font-medium text-gray-400">Tailscale IP (Optional)</label>
                <input id="ip" type="text" bind:value={newDevice.ip}
                       class="w-full bg-gray-900 border border-gray-700 rounded px-4 py-2 focus:ring-2 focus:ring-cyan-500 focus:border-transparent outline-none transition-all"
                       placeholder="100.x.y.z" />
            </div>

            <div class="grid grid-cols-2 gap-4">
                <div class="space-y-2">
                    <label for="ssh_user" class="block text-sm font-medium text-gray-400">SSH User</label>
                    <input id="ssh_user" type="text" bind:value={newDevice.ssh_user} required
                           class="w-full bg-gray-900 border border-gray-700 rounded px-4 py-2 focus:ring-2 focus:ring-cyan-500 focus:border-transparent outline-none transition-all" />
                </div>
                <div class="space-y-2">
                    <label for="ssh_port" class="block text-sm font-medium text-gray-400">SSH Port</label>
                    <input id="ssh_port" type="number" bind:value={newDevice.ssh_port} required
                           class="w-full bg-gray-900 border border-gray-700 rounded px-4 py-2 focus:ring-2 focus:ring-cyan-500 focus:border-transparent outline-none transition-all" />
                </div>
            </div>

            <div class="md:col-span-2 flex justify-end">
                <button type="submit" 
                        class="bg-gradient-to-r from-cyan-500 to-blue-600 hover:from-cyan-400 hover:to-blue-500 text-white font-bold py-2 px-6 rounded shadow-lg shadow-cyan-500/20 transition-all transform hover:scale-105">
                    Add Device
                </button>
            </div>
        </form>
    </section>

    <!-- Configure Schedule (Placeholder) -->
    <section class="opacity-75 grayscale hover:grayscale-0 transition-all duration-500">
        <h2 class="text-2xl font-bold mb-4 text-gray-400">Schedule Configuration (Future)</h2>
        <div class="bg-gray-800/30 border border-dashed border-gray-700 rounded-xl p-8 flex items-center justify-center text-gray-500">
            Schedule configuration is currently managed via config.yaml or hardcoded defaults.
        </div>
    </section>
</div>