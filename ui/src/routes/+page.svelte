<script>
    import { onMount, onDestroy } from 'svelte';
    import { getDevices, getResults } from '$lib/api';

    /** @type {import('$lib/api').Device[]} */
    let devices = [];
    /** @type {import('$lib/api').Result[]} */
    let results = [];
    let loading = true;
    /** @type {string|null} */
    let error = null;
    
    // Realtime status
    let statusMessage = "Idle";
    /** @type {string|null} */
    let lastEventTime = null;

    async function load() {
        try {
            const [d, r] = await Promise.all([getDevices(), getResults()]);
            devices = d || [];
            results = r || [];
            error = null;
        } catch (/** @type {any} */ e) {
            console.error("Initial load failed", e);
            if (loading) error = e.message;
        } finally {
            loading = false;
        }
    }

    onMount(() => {
        load();
        
        const evtSource = new EventSource('/api/events');
        evtSource.onmessage = (event) => {
            try {
                const msg = JSON.parse(event.data);
                lastEventTime = new Date().toLocaleTimeString();

                if (msg.type === 'result') {
                    const newResult = msg.data;
                    const index = results.findIndex(r => 
                        r.source_id === newResult.source_id && 
                        r.target_id === newResult.target_id && 
                        r.type === newResult.type
                    );
                    if (index !== -1) {
                        results[index] = newResult;
                    } else {
                        results.push(newResult);
                    }
                    results = results; // trigger reactivity
                    statusMessage = `Completed ${newResult.type} test: ${newResult.source_id} -> ${newResult.target_id}`;
                } else if (msg.type === 'status') {
                    statusMessage = msg.data;
                }
            } catch (e) {
                console.error("Failed to parse event", e);
            }
        };

        evtSource.onerror = (err) => {
            console.error("EventSource failed:", err);
            statusMessage = "Reconnecting to server...";
        };

        return () => {
            evtSource.close();
        };
    });

    /**
     * @param {number} sourceId
     * @param {number} targetId
     */
    function getResult(sourceId, targetId) {
        const ping = results.find(r => r.source_id === sourceId && r.target_id === targetId && r.type === 'ping');
        const speed = results.find(r => r.source_id === sourceId && r.target_id === targetId && r.type === 'speed');
        return { ping, speed };
    }
</script>

<div class="space-y-8">
    <!-- Status Line -->
    <div class="bg-gray-800 border-l-4 border-cyan-500 px-6 py-4 rounded-r-xl shadow-lg flex items-center justify-between backdrop-blur">
        <div class="flex items-center gap-4">
            <div class="relative flex h-3 w-3">
                <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-cyan-400 opacity-75"></span>
                <span class="relative inline-flex rounded-full h-3 w-3 bg-cyan-500"></span>
            </div>
            <div>
                <p class="text-xs uppercase tracking-widest text-gray-500 font-bold">Current Activity</p>
                <p class="text-white font-medium">{statusMessage}</p>
            </div>
        </div>
        {#if lastEventTime}
            <div class="text-right text-[10px] text-gray-500 font-mono">
                LAST EVENT: {lastEventTime}
            </div>
        {/if}
    </div>

    <header>
        <h1 class="text-3xl font-bold bg-gradient-to-r from-white to-gray-400 bg-clip-text text-transparent">
            Network Status
        </h1>
        <p class="text-gray-400 mt-2">Real-time connectivity matrix between nodes.</p>
    </header>

    {#if loading}
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            <div class="h-64 bg-gray-800/50 rounded-xl animate-pulse col-span-full"></div>
        </div>
    {:else if error}
        <div class="p-4 bg-red-900/50 border border-red-800 text-red-200 rounded">
            Error: {error}
        </div>
    {:else}
        <!-- Connectivity Matrix -->
        <div class="grid grid-cols-1 gap-6">
            <div class="bg-gray-800/30 border border-gray-700 rounded-2xl p-6 backdrop-blur shadow-xl">
                <h2 class="text-xl font-semibold mb-6 flex items-center text-white">
                    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-5 h-5 mr-2 text-cyan-400">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6A2.25 2.25 0 0 1 6 3.75h2.25A2.25 2.25 0 0 1 10.5 6v2.25a2.25 2.25 0 0 1-2.25 2.25H6a2.25 2.25 0 0 1-2.25-2.25V6ZM3.75 15.75A2.25 2.25 0 0 1 6 13.5h2.25a2.25 2.25 0 0 1 2.25 2.25V18a2.25 2.25 0 0 1-2.25 2.25H6A2.25 2.25 0 0 1 3.75 18v-2.25ZM13.5 6a2.25 2.25 0 0 1 2.25-2.25H18A2.25 2.25 0 0 1 20.25 6v2.25A2.25 2.25 0 0 1 18 10.5h-2.25a2.25 2.25 0 0 1-2.25-2.25V6ZM13.5 15.75a2.25 2.25 0 0 1 2.25-2.25H18a2.25 2.25 0 0 1 2.25 2.25V18a2.25 2.25 0 0 1-2.25 2.25H18a2.25 2.25 0 0 1-2.25-2.25v-2.25Z" />
                    </svg>
                    Live Connectivity
                </h2>
                
                <div class="overflow-x-auto">
                    <table class="w-full text-left border-separate border-spacing-2">
                        <thead>
                            <tr>
                                <th class="p-4 text-gray-500 text-[10px] uppercase font-black tracking-widest">Source \ Target</th>
                                {#each devices as device}
                                    <th class="p-4 font-bold text-gray-300 text-center bg-gray-800/50 rounded-lg">{device.name}</th>
                                {/each}
                            </tr>
                        </thead>
                        <tbody>
                            {#each devices as source}
                                <tr>
                                    <td class="p-4 font-bold text-gray-300 bg-gray-800/50 rounded-lg">{source.name}</td>
                                    {#each devices as target}
                                        <td class="p-4 text-center bg-gray-900/40 rounded-lg border border-gray-800/50 hover:border-cyan-500/30 transition-all group">
                                            {#if source.id === target.id}
                                                <div class="flex flex-col items-center opacity-20">
                                                    <div class="w-8 h-8 rounded-full border-2 border-dashed border-gray-600"></div>
                                                </div>
                                            {:else}
                                                {@const res = getResult(source.id, target.id)}
                                                <div class="flex flex-col gap-2">
                                                    <!-- PING RESULT -->
                                                    {#if res.ping}
                                                        {#if res.ping.error}
                                                            <div class="text-red-400 text-[10px] font-bold bg-red-900/20 py-1 rounded cursor-help" title={res.ping.error}>
                                                                PING FAIL
                                                            </div>
                                                        {:else}
                                                            <div class="flex flex-col">
                                                                <span class="text-lg font-mono font-bold text-cyan-400 leading-none">{res.ping.latency_ms.toFixed(1)}</span>
                                                                <span class="text-[9px] text-gray-600 font-bold uppercase tracking-tighter">ms latency</span>
                                                            </div>
                                                        {/if}
                                                    {:else}
                                                        <div class="text-[10px] text-gray-700 font-bold">WAITING...</div>
                                                    {/if}
                                                    
                                                    <!-- SPEED RESULT -->
                                                    {#if res.speed}
                                                        {#if res.speed.error}
                                                            <div class="text-red-400 text-[10px] font-bold bg-red-900/20 py-1 rounded cursor-help" title={res.speed.error}>
                                                                SPD FAIL
                                                            </div>
                                                        {:else}
                                                            <div class="flex items-center justify-center gap-1 bg-cyan-500/5 py-1 rounded border border-cyan-500/10">
                                                                <span class="text-xs font-bold text-gray-400">{res.speed.bandwidth_mbps.toFixed(1)}</span>
                                                                <span class="text-[8px] text-gray-600 font-black">MBPS</span>
                                                            </div>
                                                        {/if}
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
        <section class="mt-12">
            <h2 class="text-xl font-bold mb-6 flex items-center text-white">
                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-5 h-5 mr-2 text-cyan-400">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M5.25 14.25h13.5m-13.5 0a3 3 0 0 1-3-3V7.5a3 3 0 0 1 3-3h13.5a3 3 0 0 1 3 3v3.75a3 3 0 0 1-3 3m-13.5 0h13.5m-13.5 0a3 3 0 0 0-3 3v3.75a3 3 0 0 0 3 3h13.5a3 3 0 0 0 3-3v-3.75a3 3 0 0 0-3-3" />
                </svg>
                Inventory
            </h2>
            <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
                {#each devices as dev}
                    <div class="p-5 bg-gray-800/40 rounded-xl border border-gray-700 hover:border-cyan-500/40 transition-all group relative overflow-hidden">
                        <div class="absolute top-0 right-0 p-2 opacity-5">
                            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-16 h-16">
                                <path stroke-linecap="round" stroke-linejoin="round" d="M5.25 14.25h13.5m-13.5 0a3 3 0 0 1-3-3V7.5a3 3 0 0 1 3-3h13.5a3 3 0 0 1 3 3v3.75a3 3 0 0 1-3 3m-13.5 0h13.5m-13.5 0a3 3 0 0 0-3 3v3.75a3 3 0 0 0 3 3h13.5a3 3 0 0 0 3-3v-3.75a3 3 0 0 0-3-3" />
                            </svg>
                        </div>
                        <h3 class="font-bold text-white group-hover:text-cyan-400 transition-colors">{dev.name}</h3>
                        <p class="text-xs text-gray-500 font-mono mt-1">{dev.hostname}</p>
                        <div class="mt-4 flex items-center justify-between">
                            <span class="text-[10px] font-black text-gray-600 uppercase tracking-widest">{dev.ssh_user}</span>
                            <div class="w-2 h-2 rounded-full bg-green-500 shadow-[0_0_8px_rgba(34,197,94,0.4)]"></div>
                        </div>
                    </div>
                {/each}
            </div>
        </section>
    {/if}
</div>
