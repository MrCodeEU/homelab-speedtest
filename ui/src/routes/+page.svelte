<script>
    import { onMount } from 'svelte';
    import { getDevices, getResults } from '$lib/api';

    /** @type {import('$lib/api').Device[]} */
    let devices = [];
    /** @type {import('$lib/api').Result[]} */
    let results = [];
    let loading = true;
    /** @type {string|null} */
    let error = null;

    onMount(async () => {
        try {
            const [d, r] = await Promise.all([getDevices(), getResults()]);
            devices = d;
            results = r;
        } catch (/** @type {any} */ e) {
            error = e.message;
        } finally {
            loading = false;
        }
    });

    /**
     * @param {number} sourceId
     * @param {number} targetId
     */
    function getResult(sourceId, targetId) {
        // We might have two results (ping and speed).
        // Let's find them.
        const ping = results.find(r => r.source_id === sourceId && r.target_id === targetId && r.type === 'ping');
        const speed = results.find(r => r.source_id === sourceId && r.target_id === targetId && r.type === 'speed');
        return { ping, speed };
    }
</script>

<div class="space-y-8">
    <header>
        <h1 class="text-3xl font-bold bg-gradient-to-r from-white to-gray-400 bg-clip-text text-transparent">
            Network Status
        </h1>
        <p class="text-gray-400 mt-2">Real-time connectivity matrix between nodes.</p>
    </header>

    {#if loading}
        <div class="animate-pulse flex space-x-4">
            <div class="h-12 w-full bg-gray-800 rounded"></div>
        </div>
    {:else if error}
        <div class="p-4 bg-red-900/50 border border-red-800 text-red-200 rounded">
            Error: {error}
        </div>
    {:else}
        <!-- Connectivity Matrix -->
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            <div class="col-span-1 md:col-span-2 lg:col-span-3 bg-gray-800/50 border border-gray-700 rounded-xl p-6 backdrop-blur">
                <h2 class="text-xl font-semibold mb-6 flex items-center">
                    <span class="w-2 h-2 rounded-full bg-green-500 mr-2"></span>
                    Live Connectivity
                </h2>
                
                <div class="overflow-x-auto">
                    <table class="w-full text-left border-collapse">
                        <thead>
                            <tr>
                                <th class="p-4 border-b border-gray-700 text-gray-400 font-medium">Source \ Target</th>
                                {#each devices as device}
                                    <th class="p-4 border-b border-gray-700 font-medium text-white">{device.name}</th>
                                {/each}
                            </tr>
                        </thead>
                        <tbody>
                            {#each devices as source}
                                <tr class="hover:bg-gray-800/30 transition-colors">
                                    <td class="p-4 border-b border-gray-800 font-medium text-white">{source.name}</td>
                                    {#each devices as target}
                                        <td class="p-4 border-b border-gray-800">
                                            {#if source.id === target.id}
                                                <span class="text-gray-600">-</span>
                                            {:else}
                                                {@const res = getResult(source.id, target.id)}
                                                <div class="flex flex-col gap-1">
                                                    <!-- PING RESULT -->
                                                    {#if res.ping}
                                                        {#if res.ping.error}
                                                            <div class="text-red-400 text-xs font-mono flex items-center gap-1 cursor-help" title={res.ping.error}>
                                                                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="w-3 h-3">
                                                                    <path fill-rule="evenodd" d="M18 10a8 8 0 1 1-16 0 8 8 0 0 1 16 0Zm-8-5a.75.75 0 0 1 .75.75v4.5a.75.75 0 0 1-1.5 0v-4.5A.75.75 0 0 1 10 5Zm0 10a1 1 0 1 0 0-2 1 1 0 0 0 0 2Z" clip-rule="evenodd" />
                                                                </svg>
                                                                Ping Err
                                                            </div>
                                                        {:else}
                                                            <span class="text-sm font-mono text-cyan-400">{res.ping.latency_ms.toFixed(1)}ms</span>
                                                        {/if}
                                                    {:else}
                                                        <span class="text-sm font-mono text-gray-600">-- ms</span>
                                                    {/if}
                                                    
                                                    <!-- SPEED RESULT -->
                                                    {#if res.speed}
                                                        {#if res.speed.error}
                                                            <div class="text-red-400 text-xs font-mono flex items-center gap-1 cursor-help" title={res.speed.error}>
                                                                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="w-3 h-3">
                                                                    <path fill-rule="evenodd" d="M18 10a8 8 0 1 1-16 0 8 8 0 0 1 16 0Zm-8-5a.75.75 0 0 1 .75.75v4.5a.75.75 0 0 1-1.5 0v-4.5A.75.75 0 0 1 10 5Zm0 10a1 1 0 1 0 0-2 1 1 0 0 0 0 2Z" clip-rule="evenodd" />
                                                                </svg>
                                                                Spd Err
                                                            </div>
                                                        {:else}
                                                            <span class="text-xs text-gray-500">{res.speed.bandwidth_mbps.toFixed(1)} Mbps</span>
                                                        {/if}
                                                    {:else}
                                                        <span class="text-xs text-gray-600">-- Mbps</span>
                                                    {/if}
                                                </div>
                                            {/if}
                                        </td>
                                    {/each}
                                </tr>
                            {/each}
                        </tbody>
                    </table>
                </div>
            </div>
        </div>

        <!-- Devices List -->
        <section>
            <h2 class="text-2xl font-bold mb-4">Devices</h2>
            <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
                {#each devices as dev}
                    <div class="p-6 bg-gray-800 rounded-xl border border-gray-700 hover:border-cyan-500/50 transition-colors group">
                        <div class="flex justify-between items-start">
                            <div>
                                <h3 class="font-bold text-lg group-hover:text-cyan-400 transition-colors">{dev.name}</h3>
                                <div class="text-sm text-gray-400 mt-1">{dev.hostname}</div>
                                <div class="text-xs text-gray-500 mt-2 font-mono bg-gray-900 px-2 py-1 rounded inline-block">
                                    {dev.ssh_user}@{dev.ip || dev.hostname}:{dev.ssh_port}
                                </div>
                            </div>
                            <div class="w-2 h-2 rounded-full bg-green-500 shadow-[0_0_10px_rgba(34,197,94,0.5)]"></div>
                        </div>
                    </div>
                {/each}
            </div>
        </section>
    {/if}
</div>