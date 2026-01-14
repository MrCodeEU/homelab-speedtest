<script>
    import { onMount } from 'svelte';
    import { getDevices, addDevice } from '$lib/api';

    /** @type {import('$lib/api').Device[]} */
    let devices = [];
    let loading = true;
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
        } catch (e) {
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
        } catch (e) {
            alert('Failed to add device: ' + e.message);
        }
    }
</script>

<div class="max-w-4xl mx-auto space-y-12">
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
