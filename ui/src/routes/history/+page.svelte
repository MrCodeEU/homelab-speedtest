<script>
    import { onMount } from 'svelte';
    import { getHistory, getDevices } from '$lib/api';
    import Chart from 'chart.js/auto';

    /** @type {import('$lib/api').Result[]} */
    let history = [];
    /** @type {import('$lib/api').Device[]} */
    let devices = [];
    let loading = true;
    /** @type {string|null} */
    let error = null;

    /** @type {HTMLCanvasElement} */
    let latencyCanvas;
    /** @type {HTMLCanvasElement} */
    let bandwidthCanvas;
    /** @type {Chart} */
    let latencyChart;
    /** @type {Chart} */
    let bandwidthChart;

    async function load() {
        try {
            const [h, d] = await Promise.all([getHistory(200), getDevices()]);
            history = h;
            devices = d || [];
            renderCharts();
        } catch (e) {
            error = e instanceof Error ? e.message : String(e);
        } finally {
            loading = false;
        }
    }

    /**
     * @param {number} id
     */
    function getDeviceName(id) {
        const d = devices.find(x => x.id === id);
        return d ? d.name : `Device ${id}`;
    }

    function renderCharts() {
        if (!latencyCanvas || !bandwidthCanvas) return;

        /** 
         * @type {Record<string, {
         *   pings: {x: string, y: number}[], 
         *   speeds: {x: string, y: number}[]
         * }>} 
         */
        const pairs = {};
        
        // Sort by timestamp asc for chart
        const sorted = [...history].reverse();

        sorted.forEach(r => {
            const key = `${getDeviceName(r.source_id)} -> ${getDeviceName(r.target_id)}`;
            if (!pairs[key]) pairs[key] = { pings: [], speeds: [] };
            
            if (r.type === 'ping') {
                pairs[key].pings.push({ x: r.timestamp, y: r.latency_ms });
            } else if (r.type === 'speed') {
                pairs[key].speeds.push({ x: r.timestamp, y: r.bandwidth_mbps });
            }
        });

        const colors = [
            '#06b6d4', // cyan-500
            '#22c55e', // green-500
            '#eab308', // yellow-500
            '#f43f5e', // rose-500
            '#8b5cf6', // violet-500
        ];
        let colorIdx = 0;

        /** @type {import('chart.js').ChartDataset[]} */
        const latencyDatasets = [];
        /** @type {import('chart.js').ChartDataset[]} */
        const speedDatasets = [];

        Object.keys(pairs).forEach(key => {
            const color = colors[colorIdx % colors.length];
            colorIdx++;

            if (pairs[key].pings.length > 0) {
                latencyDatasets.push({
                    type: 'line',
                    label: key,
                    data: /** @type {any} */ (pairs[key].pings),
                    borderColor: color,
                    tension: 0.1
                });
            }
            if (pairs[key].speeds.length > 0) {
                speedDatasets.push({
                    type: 'line',
                    label: key,
                    data: /** @type {any} */ (pairs[key].speeds),
                    borderColor: color,
                    tension: 0.1
                });
            }
        });

        if (latencyChart) latencyChart.destroy();
        latencyChart = new Chart(latencyCanvas, {
            type: 'line',
            data: { datasets: latencyDatasets },
            options: {
                responsive: true,
                interaction: { mode: 'index', intersect: false },
                plugins: {
                    title: { display: true, text: 'Latency (ms)', color: '#fff' },
                    legend: { labels: { color: '#aaa' } }
                },
                scales: {
                    x: { type: 'category', ticks: { color: '#777', maxTicksLimit: 10 } },
                    y: { ticks: { color: '#777' }, grid: { color: '#333' } }
                }
            }
        });

        if (bandwidthChart) bandwidthChart.destroy();
        bandwidthChart = new Chart(bandwidthCanvas, {
            type: 'line',
            data: { datasets: speedDatasets },
            options: {
                responsive: true,
                interaction: { mode: 'index', intersect: false },
                plugins: {
                    title: { display: true, text: 'Bandwidth (Mbps)', color: '#fff' },
                    legend: { labels: { color: '#aaa' } }
                },
                scales: {
                    x: { type: 'category', ticks: { color: '#777', maxTicksLimit: 10 } },
                    y: { ticks: { color: '#777' }, grid: { color: '#333' } }
                }
            }
        });
    }

    onMount(() => {
        load();
        return () => {
            if (latencyChart) latencyChart.destroy();
            if (bandwidthChart) bandwidthChart.destroy();
        };
    });
</script>

<div class="space-y-8">
    <header>
        <h1 class="text-3xl font-bold text-white">History</h1>
        <p class="text-gray-400 mt-2">Historical performance data.</p>
    </header>

    {#if loading}
        <div class="text-gray-400">Loading history...</div>
    {:else if error}
        <div class="text-red-400">Error: {error}</div>
    {:else}
        <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div class="bg-gray-800/50 border border-gray-700 rounded-xl p-4 backdrop-blur">
                <canvas bind:this={latencyCanvas}></canvas>
            </div>
            <div class="bg-gray-800/50 border border-gray-700 rounded-xl p-4 backdrop-blur">
                <canvas bind:this={bandwidthCanvas}></canvas>
            </div>
        </div>

        <div class="bg-gray-800/50 border border-gray-700 rounded-xl overflow-hidden backdrop-blur">
            <div class="overflow-x-auto">
                <table class="w-full text-left text-sm text-gray-400">
                    <thead class="bg-gray-900 text-gray-200 uppercase font-medium">
                        <tr>
                            <th class="p-3">Time</th>
                            <th class="p-3">Type</th>
                            <th class="p-3">Source -> Target</th>
                            <th class="p-3 text-right">Value</th>
                            <th class="p-3">Status</th>
                        </tr>
                    </thead>
                    <tbody class="divide-y divide-gray-700">
                        {#each history as item}
                            <tr class="hover:bg-gray-800/50">
                                <td class="p-3 font-mono">{new Date(item.timestamp).toLocaleString()}</td>
                                <td class="p-3 uppercase text-xs font-bold tracking-wide">
                                    <span class={item.type === 'ping' ? 'text-cyan-400' : 'text-green-400'}>
                                        {item.type}
                                    </span>
                                </td>
                                <td class="p-3">
                                    {getDeviceName(item.source_id)} <span class="text-gray-600">→</span> {getDeviceName(item.target_id)}
                                </td>
                                <td class="p-3 text-right font-mono text-white">
                                    {#if item.type === 'ping'}
                                        {item.latency_ms.toFixed(2)} ms
                                    {:else}
                                        {item.bandwidth_mbps.toFixed(2)} Mbps
                                    {/if}
                                </td>
                                <td class="p-3">
                                    {#if item.error}
                                        <span class="text-red-400" title={item.error}>Failed</span>
                                    {:else}
                                        <span class="text-green-500">Success</span>
                                    {/if}
                                </td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
            </div>
        </div>
    {/if}
</div>
