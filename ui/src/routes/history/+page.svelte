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

    // Time range selection
    let selectedRange = '24h';
    const timeRanges = [
        { value: '1h', label: '1 Hour' },
        { value: '6h', label: '6 Hours' },
        { value: '24h', label: '24 Hours' },
        { value: '7d', label: '7 Days' },
        { value: '30d', label: '30 Days' },
        { value: 'all', label: 'All Time' }
    ];

    /** @type {HTMLCanvasElement} */
    let latencyCanvas;
    /** @type {HTMLCanvasElement} */
    let bandwidthCanvas;
    /** @type {Chart|null} */
    let latencyChart = null;
    /** @type {Chart|null} */
    let bandwidthChart = null;

    async function load() {
        try {
            const [h, d] = await Promise.all([getHistory(500), getDevices()]);
            history = h || [];
            devices = d || [];
            error = null;
            // Render charts after data loads
            setTimeout(renderCharts, 50);
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

    /**
     * Get filtered history based on time range
     */
    function getFilteredHistory() {
        if (selectedRange === 'all') return history;

        const now = new Date();
        let cutoff = new Date();

        switch (selectedRange) {
            case '1h': cutoff.setHours(now.getHours() - 1); break;
            case '6h': cutoff.setHours(now.getHours() - 6); break;
            case '24h': cutoff.setDate(now.getDate() - 1); break;
            case '7d': cutoff.setDate(now.getDate() - 7); break;
            case '30d': cutoff.setDate(now.getDate() - 30); break;
        }

        return history.filter(r => new Date(r.timestamp) >= cutoff);
    }

    function renderCharts() {
        if (!latencyCanvas || !bandwidthCanvas) {
            console.log("Canvas not ready");
            return;
        }

        const filteredHistory = getFilteredHistory();
        console.log("Rendering charts with", filteredHistory.length, "records");

        if (filteredHistory.length === 0) {
            // Clear charts if no data
            if (latencyChart) {
                latencyChart.data.labels = [];
                latencyChart.data.datasets = [];
                latencyChart.update();
            }
            if (bandwidthChart) {
                bandwidthChart.data.labels = [];
                bandwidthChart.data.datasets = [];
                bandwidthChart.update();
            }
            return;
        }

        // Group data by device pair
        /** @type {Record<string, {pings: {time: string, value: number}[], speeds: {time: string, value: number}[]}>} */
        const pairs = {};

        // Sort by timestamp ascending for chart
        const sorted = [...filteredHistory].sort((a, b) =>
            new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime()
        );

        sorted.forEach(r => {
            if (r.error) return; // Skip failed results

            const key = `${getDeviceName(r.source_id)} → ${getDeviceName(r.target_id)}`;
            if (!pairs[key]) pairs[key] = { pings: [], speeds: [] };

            const time = new Date(r.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });

            if (r.type === 'ping' && r.latency_ms > 0) {
                pairs[key].pings.push({ time, value: r.latency_ms });
            } else if (r.type === 'speed' && r.bandwidth_mbps > 0) {
                pairs[key].speeds.push({ time, value: r.bandwidth_mbps });
            }
        });

        const colors = [
            '#06b6d4', // cyan-500
            '#22c55e', // green-500
            '#eab308', // yellow-500
            '#f43f5e', // rose-500
            '#8b5cf6', // violet-500
            '#ec4899', // pink-500
            '#f97316', // orange-500
            '#14b8a6', // teal-500
        ];

        // Build latency datasets
        /** @type {any[]} */
        const latencyDatasets = [];
        /** @type {string[]} */
        let latencyLabels = [];
        let colorIdx = 0;

        Object.entries(pairs).forEach(([key, data]) => {
            if (data.pings.length > 0) {
                const color = colors[colorIdx % colors.length];
                colorIdx++;

                // Use the timestamps from this pair as labels (merge later)
                if (data.pings.length > latencyLabels.length) {
                    latencyLabels = data.pings.map(p => p.time);
                }

                latencyDatasets.push({
                    label: key,
                    data: data.pings.map(p => p.value),
                    borderColor: color,
                    backgroundColor: color + '20',
                    tension: 0.3,
                    fill: false,
                    pointRadius: 2
                });
            }
        });

        // Build bandwidth datasets
        /** @type {any[]} */
        const speedDatasets = [];
        /** @type {string[]} */
        let speedLabels = [];
        colorIdx = 0;

        Object.entries(pairs).forEach(([key, data]) => {
            if (data.speeds.length > 0) {
                const color = colors[colorIdx % colors.length];
                colorIdx++;

                if (data.speeds.length > speedLabels.length) {
                    speedLabels = data.speeds.map(p => p.time);
                }

                speedDatasets.push({
                    label: key,
                    data: data.speeds.map(p => p.value),
                    borderColor: color,
                    backgroundColor: color + '20',
                    tension: 0.3,
                    fill: false,
                    pointRadius: 2
                });
            }
        });

        // Destroy and recreate charts
        if (latencyChart) latencyChart.destroy();
        latencyChart = new Chart(latencyCanvas, {
            type: 'line',
            data: {
                labels: latencyLabels,
                datasets: latencyDatasets
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                interaction: { mode: 'index', intersect: false },
                plugins: {
                    title: { display: true, text: 'Latency (ms)', color: '#fff', font: { size: 14 } },
                    legend: {
                        position: 'bottom',
                        labels: { color: '#aaa', boxWidth: 12, padding: 15 }
                    }
                },
                scales: {
                    x: {
                        ticks: { color: '#666', maxTicksLimit: 10, maxRotation: 45 },
                        grid: { color: '#333' }
                    },
                    y: {
                        ticks: { color: '#666' },
                        grid: { color: '#333' },
                        beginAtZero: true
                    }
                }
            }
        });

        if (bandwidthChart) bandwidthChart.destroy();
        bandwidthChart = new Chart(bandwidthCanvas, {
            type: 'line',
            data: {
                labels: speedLabels,
                datasets: speedDatasets
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                interaction: { mode: 'index', intersect: false },
                plugins: {
                    title: { display: true, text: 'Bandwidth (Mbps)', color: '#fff', font: { size: 14 } },
                    legend: {
                        position: 'bottom',
                        labels: { color: '#aaa', boxWidth: 12, padding: 15 }
                    }
                },
                scales: {
                    x: {
                        ticks: { color: '#666', maxTicksLimit: 10, maxRotation: 45 },
                        grid: { color: '#333' }
                    },
                    y: {
                        ticks: { color: '#666' },
                        grid: { color: '#333' },
                        beginAtZero: true
                    }
                }
            }
        });
    }

    function handleRangeChange() {
        renderCharts();
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
    <header class="flex flex-col md:flex-row md:items-end justify-between gap-4">
        <div>
            <h1 class="text-3xl font-bold text-white">History</h1>
            <p class="text-gray-400 mt-2">Historical performance data.</p>
        </div>

        <!-- Time Range Selector -->
        <div class="flex items-center gap-2">
            <span class="text-sm text-gray-400">Time Range:</span>
            <div class="flex bg-gray-800 rounded-lg p-1">
                {#each timeRanges as range}
                    <button
                        onclick={() => { selectedRange = range.value; handleRangeChange(); }}
                        class="px-3 py-1.5 text-sm font-medium rounded-md transition-all {selectedRange === range.value ? 'bg-cyan-600 text-white' : 'text-gray-400 hover:text-white'}"
                    >
                        {range.label}
                    </button>
                {/each}
            </div>
        </div>
    </header>

    {#if loading}
        <div class="text-gray-400 animate-pulse">Loading history...</div>
    {:else if error}
        <div class="p-4 bg-red-900/50 border border-red-800 text-red-200 rounded">Error: {error}</div>
    {:else if history.length === 0}
        <div class="p-8 bg-gray-800/50 border border-gray-700 rounded-xl text-center">
            <p class="text-gray-400">No history data available yet.</p>
            <p class="text-gray-500 text-sm mt-2">Run some tests to see data here.</p>
        </div>
    {:else}
        <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div class="bg-gray-800/50 border border-gray-700 rounded-xl p-4 backdrop-blur">
                <div class="h-72">
                    <canvas bind:this={latencyCanvas}></canvas>
                </div>
            </div>
            <div class="bg-gray-800/50 border border-gray-700 rounded-xl p-4 backdrop-blur">
                <div class="h-72">
                    <canvas bind:this={bandwidthCanvas}></canvas>
                </div>
            </div>
        </div>

        <div class="bg-gray-800/50 border border-gray-700 rounded-xl overflow-hidden backdrop-blur">
            <div class="px-4 py-3 border-b border-gray-700 flex justify-between items-center">
                <h3 class="text-sm font-semibold text-gray-300">Test Results ({getFilteredHistory().length} records)</h3>
            </div>
            <div class="overflow-x-auto max-h-96">
                <table class="w-full text-left text-sm text-gray-400">
                    <thead class="bg-gray-900 text-gray-200 uppercase font-medium sticky top-0">
                        <tr>
                            <th class="p-3">Time</th>
                            <th class="p-3">Type</th>
                            <th class="p-3">Source → Target</th>
                            <th class="p-3 text-right">Value</th>
                            <th class="p-3">Status</th>
                        </tr>
                    </thead>
                    <tbody class="divide-y divide-gray-700">
                        {#each getFilteredHistory() as item}
                            <tr class="hover:bg-gray-800/50">
                                <td class="p-3 font-mono text-xs">{new Date(item.timestamp).toLocaleString()}</td>
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
                                        <span class="text-red-400 cursor-help" title={item.error}>Failed</span>
                                    {:else}
                                        <span class="text-green-500">OK</span>
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
