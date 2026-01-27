<script>
    import { onMount, onDestroy } from 'svelte';
    import { getDevices, getResults, getScheduleStatus, triggerPingAll, triggerSpeedAll, getHistory, getQueueStatus } from '$lib/api';
    import Chart from 'chart.js/auto';

    /** @type {import('$lib/api').Device[]} */
    let devices = [];
    /** @type {import('$lib/api').Result[]} */
    let results = [];
    /** @type {import('$lib/api').ScheduleStatus[]} */
    let scheduleStatus = [];
    /** @type {import('$lib/api').Result[]} */
    let history = [];
    /** @type {import('$lib/api').QueueStatus|null} */
    let queueStatus = null;
    let loading = true;
    /** @type {string|null} */
    let error = null;

    // Realtime status
    let statusMessage = "Idle";
    /** @type {string|null} */
    let lastEventTime = null;
    let connected = false;

    // Action states
    let pingingAll = false;
    let speedingAll = false;

    // Time range selection
    let selectedRange = '1h';
    const timeRanges = [
        { value: '1h', label: '1 Hour' },
        { value: '6h', label: '6 Hours' },
        { value: '24h', label: '24 Hours' },
        { value: '7d', label: '7 Days' },
        { value: '30d', label: '30 Days' }
    ];

    // Chart references
    /** @type {HTMLCanvasElement} */
    let latencyCanvas;
    /** @type {HTMLCanvasElement} */
    let bandwidthCanvas;
    /** @type {Chart|null} */
    let latencyChart = null;
    /** @type {Chart|null} */
    let bandwidthChart = null;

    // Chart colors for different device pairs
    const chartColors = [
        '#06b6d4', // cyan-500
        '#22c55e', // green-500
        '#eab308', // yellow-500
        '#f43f5e', // rose-500
        '#8b5cf6', // violet-500
        '#ec4899', // pink-500
        '#f97316', // orange-500
        '#14b8a6', // teal-500
    ];

    // Countdown timer
    let countdownInterval = null;
    let pingCountdown = "";
    let speedCountdown = "";

    function updateCountdowns() {
        const now = new Date();

        const pingSchedule = scheduleStatus.find(s => s.type === 'ping');
        const speedSchedule = scheduleStatus.find(s => s.type === 'speed');

        if (pingSchedule?.enabled && pingSchedule.next_run) {
            const nextPing = new Date(pingSchedule.next_run);
            const diff = Math.max(0, Math.floor((nextPing.getTime() - now.getTime()) / 1000));
            const mins = Math.floor(diff / 60);
            const secs = diff % 60;
            pingCountdown = `${mins}:${secs.toString().padStart(2, '0')}`;
        } else {
            pingCountdown = pingSchedule?.enabled === false ? "Disabled" : "--:--";
        }

        if (speedSchedule?.enabled && speedSchedule.next_run) {
            const nextSpeed = new Date(speedSchedule.next_run);
            const diff = Math.max(0, Math.floor((nextSpeed.getTime() - now.getTime()) / 1000));
            const mins = Math.floor(diff / 60);
            const secs = diff % 60;
            speedCountdown = `${mins}:${secs.toString().padStart(2, '0')}`;
        } else {
            speedCountdown = speedSchedule?.enabled === false ? "Disabled" : "--:--";
        }
    }

    async function runPingAll() {
        if (pingingAll) return;
        pingingAll = true;
        try {
            await triggerPingAll();
            statusMessage = "Manual ping tests initiated...";
        } catch (/** @type {any} */ e) {
            statusMessage = "Error: " + e.message;
        } finally {
            setTimeout(() => { pingingAll = false; }, 5000);
        }
    }

    async function runSpeedAll() {
        if (speedingAll) return;
        speedingAll = true;
        try {
            await triggerSpeedAll();
            statusMessage = "Manual speed tests initiated...";
        } catch (/** @type {any} */ e) {
            statusMessage = "Error: " + e.message;
        } finally {
            setTimeout(() => { speedingAll = false; }, 10000);
        }
    }

    async function load() {
        try {
            const [d, r, s, hPing, hSpeed, q] = await Promise.all([
                getDevices(),
                getResults(),
                getScheduleStatus(),
                getHistory(500, 'ping'),
                getHistory(500, 'speed'),
                getQueueStatus().catch(() => null)
            ]);
            devices = d || [];
            results = r || [];
            scheduleStatus = s || [];
            history = [...(hPing || []), ...(hSpeed || [])];
            queueStatus = q;
            error = null;
            updateCountdowns();
            // Render charts after data loads
            setTimeout(renderCharts, 50);
        } catch (/** @type {any} */ e) {
            console.error("Initial load failed", e);
            if (loading) error = e.message;
        } finally {
            loading = false;
        }
    }

    /**
     * Get filtered history based on time range
     */
    function getFilteredHistory() {
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

    function handleRangeChange() {
        renderCharts();
    }

    /**
     * @param {import('$lib/api').Result} newResult
     */
    function addToChartData(newResult) {
        // Add new result to history for real-time updates
        if (!newResult.error) {
            history = [...history, newResult];
            renderCharts();
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
        if (!latencyCanvas || !bandwidthCanvas) {
            return;
        }

        const filteredHistory = getFilteredHistory();

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

        // Build latency datasets
        /** @type {any[]} */
        const latencyDatasets = [];
        /** @type {string[]} */
        let latencyLabels = [];
        let colorIdx = 0;

        Object.entries(pairs).forEach(([key, data]) => {
            if (data.pings.length > 0) {
                const color = chartColors[colorIdx % chartColors.length];
                colorIdx++;

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
                const color = chartColors[colorIdx % chartColors.length];
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
                        labels: { color: '#aaa', boxWidth: 12, padding: 10 }
                    }
                },
                scales: {
                    x: {
                        ticks: { color: '#666', maxTicksLimit: 8, maxRotation: 45 },
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
                        labels: { color: '#aaa', boxWidth: 12, padding: 10 }
                    }
                },
                scales: {
                    x: {
                        ticks: { color: '#666', maxTicksLimit: 8, maxRotation: 45 },
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

    onMount(() => {
        load();

        // Polling fallback every 30s
        const pollInterval = setInterval(load, 30000);

        // Countdown timer update every second
        countdownInterval = setInterval(updateCountdowns, 1000);

        // WebSocket connection with auto-reconnect
        /** @type {WebSocket|null} */
        let ws = null;
        let reconnectTimeout = null;
        let reconnectAttempts = 0;
        const maxReconnectAttempts = 10;

        function handleMessage(data) {
            try {
                const msg = JSON.parse(data);
                lastEventTime = new Date().toLocaleTimeString();

                if (msg.type === 'result') {
                    const newResult = msg.data;
                    console.log("WS: Updating result matrix with:", newResult);

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
                    results = [...results];

                    // Add to chart data
                    addToChartData(newResult);

                    statusMessage = `Updated ${newResult.type} for ${getDeviceName(newResult.source_id)} → ${getDeviceName(newResult.target_id)}`;
                } else if (msg.type === 'status') {
                    statusMessage = msg.data;
                } else if (msg.type === 'schedule') {
                    scheduleStatus = msg.data;
                    updateCountdowns();
                } else if (msg.type === 'queue') {
                    queueStatus = msg.data;
                }
            } catch (e) {
                console.error("Failed to parse message", e);
            }
        }

        function connectWebSocket() {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const wsUrl = `${protocol}//${window.location.host}/api/ws`;

            console.log("Connecting to WebSocket:", wsUrl);
            ws = new WebSocket(wsUrl);

            ws.onopen = () => {
                connected = true;
                reconnectAttempts = 0;
                console.log("WebSocket connected");
                statusMessage = "Connected";
            };

            ws.onmessage = (event) => {
                handleMessage(event.data);
            };

            ws.onerror = (err) => {
                console.error("WebSocket error:", err);
            };

            ws.onclose = (event) => {
                connected = false;
                console.log("WebSocket closed:", event.code, event.reason);

                // Auto-reconnect with exponential backoff
                if (reconnectAttempts < maxReconnectAttempts) {
                    const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000);
                    statusMessage = `Reconnecting in ${Math.round(delay/1000)}s...`;
                    reconnectTimeout = setTimeout(() => {
                        reconnectAttempts++;
                        connectWebSocket();
                    }, delay);
                } else {
                    statusMessage = "Connection lost. Refresh to reconnect.";
                }
            };
        }

        connectWebSocket();

        return () => {
            if (ws) {
                ws.close();
            }
            if (reconnectTimeout) {
                clearTimeout(reconnectTimeout);
            }
            clearInterval(pollInterval);
            if (countdownInterval) clearInterval(countdownInterval);
            if (latencyChart) latencyChart.destroy();
            if (bandwidthChart) bandwidthChart.destroy();
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
    <div class="bg-gray-800 border-l-4 {connected ? 'border-cyan-500' : 'border-red-500'} px-6 py-4 rounded-r-xl shadow-lg flex items-center justify-between backdrop-blur">
        <div class="flex items-center gap-4">
            <div class="relative flex h-3 w-3">
                {#if connected}
                    <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-cyan-400 opacity-75"></span>
                    <span class="relative inline-flex rounded-full h-3 w-3 bg-cyan-500"></span>
                {:else}
                    <span class="relative inline-flex rounded-full h-3 w-3 bg-red-500"></span>
                {/if}
            </div>
            <div>
                <p class="text-xs uppercase tracking-widest text-gray-500 font-bold">
                    {connected ? 'Live Monitoring' : 'Connection Lost'}
                </p>
                <p class="text-white font-medium">{statusMessage}</p>
            </div>
        </div>
        <div class="flex items-center gap-4">
            {#if queueStatus && (queueStatus.running || queueStatus.length > 0)}
                <div class="flex items-center gap-2 px-3 py-1 bg-yellow-500/10 border border-yellow-500/30 rounded-lg">
                    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-4 h-4 text-yellow-400 animate-pulse">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M3.75 12h16.5m-16.5 3.75h16.5M3.75 19.5h16.5M5.625 4.5h12.75a1.875 1.875 0 0 1 0 3.75H5.625a1.875 1.875 0 0 1 0-3.75Z" />
                    </svg>
                    <span class="text-[10px] font-bold text-yellow-400 uppercase tracking-wider">
                        {#if queueStatus.running}
                            {queueStatus.running.type === 'ping_all' ? 'Ping' : 'Speed'} Running
                        {/if}
                        {#if queueStatus.length > 0}
                            {#if queueStatus.running}, {/if}{queueStatus.length} Queued
                        {/if}
                    </span>
                </div>
            {/if}
            {#if lastEventTime}
                <div class="text-right text-[10px] text-gray-500 font-mono">
                    LAST UPDATE: {lastEventTime}
                </div>
            {/if}
        </div>
    </div>

    <header class="flex flex-col md:flex-row md:items-end justify-between gap-4">
        <div>
            <h1 class="text-3xl font-bold bg-gradient-to-r from-white to-gray-400 bg-clip-text text-transparent">
                Network Status
            </h1>
            <p class="text-gray-400 mt-2">Real-time connectivity matrix between nodes.</p>
        </div>

        <div class="flex gap-3">
            <button
                onclick={load}
                class="p-2 bg-gray-800 border border-gray-700 rounded-lg text-gray-400 hover:text-white transition-all"
                title="Refresh Data"
            >
                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class={`w-5 h-5 ${loading ? 'animate-spin' : ''}`}>
                    <path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0 3.181 3.183a8.25 8.25 0 0 0 13.803-3.7M4.031 9.865a8.25 8.25 0 0 1 13.803-3.7l3.181 3.182m0-4.991v4.99" />
                </svg>
            </button>

            <button
                onclick={runPingAll}
                disabled={pingingAll}
                class="flex items-center gap-2 px-4 py-2 bg-gray-800 border border-gray-700 rounded-lg text-sm font-semibold text-gray-300 hover:text-white hover:border-cyan-500/50 transition-all disabled:opacity-50 disabled:cursor-not-allowed"
            >
                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class={`w-4 h-4 ${pingingAll ? 'animate-spin' : ''}`}>
                    <path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0 3.181 3.183a8.25 8.25 0 0 0 13.803-3.7M4.031 9.865a8.25 8.25 0 0 1 13.803-3.7l3.181 3.182m0-4.991v4.99" />
                </svg>
                {pingingAll ? 'Pinging...' : 'Ping All'}
            </button>

            <button
                onclick={runSpeedAll}
                disabled={speedingAll}
                class="flex items-center gap-2 px-4 py-2 bg-cyan-600/10 border border-cyan-500/30 rounded-lg text-sm font-semibold text-cyan-400 hover:bg-cyan-600/20 transition-all disabled:opacity-50 disabled:cursor-not-allowed"
            >
                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class={`w-4 h-4 ${speedingAll ? 'animate-spin' : ''}`}>
                    <path stroke-linecap="round" stroke-linejoin="round" d="M3.75 13.5l10.5-11.25L12 10.5h8.25L9.75 21.75 12 13.5H3.75z" />
                </svg>
                {speedingAll ? 'Running Tests...' : 'Speed Test All'}
            </button>
        </div>
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
        <!-- Next Scheduled Tests -->
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div class="bg-gray-800/30 border border-gray-700 rounded-xl p-5 backdrop-blur">
                <div class="flex items-center justify-between">
                    <div class="flex items-center gap-3">
                        <div class="p-2 bg-cyan-500/10 rounded-lg">
                            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-5 h-5 text-cyan-400">
                                <path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" />
                            </svg>
                        </div>
                        <div>
                            <p class="text-[10px] uppercase tracking-widest text-gray-500 font-bold">Next Ping Test</p>
                            <p class="text-xs text-gray-400">
                                {scheduleStatus.find(s => s.type === 'ping')?.interval || '1m'} interval
                            </p>
                        </div>
                    </div>
                    <div class="text-right">
                        <p class="text-2xl font-mono font-bold text-cyan-400">{pingCountdown}</p>
                    </div>
                </div>
            </div>

            <div class="bg-gray-800/30 border border-gray-700 rounded-xl p-5 backdrop-blur">
                <div class="flex items-center justify-between">
                    <div class="flex items-center gap-3">
                        <div class="p-2 bg-green-500/10 rounded-lg">
                            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-5 h-5 text-green-400">
                                <path stroke-linecap="round" stroke-linejoin="round" d="M3.75 13.5l10.5-11.25L12 10.5h8.25L9.75 21.75 12 13.5H3.75z" />
                            </svg>
                        </div>
                        <div>
                            <p class="text-[10px] uppercase tracking-widest text-gray-500 font-bold">Next Speed Test</p>
                            <p class="text-xs text-gray-400">
                                {scheduleStatus.find(s => s.type === 'speed')?.interval || '5m'} interval
                            </p>
                        </div>
                    </div>
                    <div class="text-right">
                        <p class="text-2xl font-mono font-bold text-green-400">{speedCountdown}</p>
                    </div>
                </div>
            </div>
        </div>

        <!-- Performance Charts -->
        <div class="space-y-4">
            <!-- Time Range Selector -->
            <div class="flex items-center justify-between">
                <h3 class="text-lg font-semibold text-white flex items-center gap-2">
                    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-5 h-5 text-cyan-400">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M3 13.125C3 12.504 3.504 12 4.125 12h2.25c.621 0 1.125.504 1.125 1.125v6.75C7.5 20.496 6.996 21 6.375 21h-2.25A1.125 1.125 0 0 1 3 19.875v-6.75ZM9.75 8.625c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125v11.25c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 0 1-1.125-1.125V8.625ZM16.5 4.125c0-.621.504-1.125 1.125-1.125h2.25C20.496 3 21 3.504 21 4.125v15.75c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 0 1-1.125-1.125V4.125Z" />
                    </svg>
                    Performance Charts
                </h3>
                <div class="flex items-center gap-2">
                    <span class="text-xs text-gray-500">Time Range:</span>
                    <div class="flex bg-gray-800 rounded-lg p-1">
                        {#each timeRanges as range}
                            <button
                                onclick={() => { selectedRange = range.value; handleRangeChange(); }}
                                class="px-3 py-1 text-xs font-medium rounded-md transition-all {selectedRange === range.value ? 'bg-cyan-600 text-white' : 'text-gray-400 hover:text-white'}"
                            >
                                {range.label}
                            </button>
                        {/each}
                    </div>
                </div>
            </div>

            <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <div class="bg-gray-800/30 border border-gray-700 rounded-xl p-5 backdrop-blur">
                    <div class="h-56">
                        <canvas bind:this={latencyCanvas}></canvas>
                    </div>
                </div>

                <div class="bg-gray-800/30 border border-gray-700 rounded-xl p-5 backdrop-blur">
                    <div class="h-56">
                        <canvas bind:this={bandwidthCanvas}></canvas>
                    </div>
                </div>
            </div>
        </div>

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
                                                {#if !res.ping && !res.speed}
                                                    <div class="text-[10px] text-gray-700 font-bold animate-pulse">WAITING...</div>
                                                {:else}
                                                    <div class="flex flex-col gap-2">
                                                        <!-- PING RESULT -->
                                                        {#if res.ping}
                                                            {#if res.ping.error}
                                                                <div class="text-red-400 text-[10px] font-bold bg-red-900/20 py-1 rounded cursor-help border border-red-500/30" title={res.ping.error}>
                                                                    PING FAIL
                                                                </div>
                                                            {:else}
                                                                <div class="flex flex-col">
                                                                    <span class="text-lg font-mono font-bold text-cyan-400 leading-none">{res.ping.latency_ms.toFixed(1)}</span>
                                                                    <span class="text-[9px] text-gray-600 font-bold uppercase tracking-tighter">ms latency</span>
                                                                </div>
                                                            {/if}
                                                        {/if}

                                                        <!-- SPEED RESULT -->
                                                        {#if res.speed}
                                                            {#if res.speed.error}
                                                                <div class="text-red-400 text-[10px] font-bold bg-red-900/20 py-1 rounded cursor-help border border-red-500/30" title={res.speed.error}>
                                                                    SPD FAIL
                                                                    <div class="text-[8px] font-normal opacity-70 truncate px-1">{res.speed.error}</div>
                                                                </div>
                                                            {:else}
                                                                <div class="flex items-center justify-center gap-1 bg-cyan-500/10 py-1.5 rounded border border-cyan-500/20">
                                                                    <span class="text-xs font-bold text-white">{res.speed.bandwidth_mbps.toFixed(1)}</span>
                                                                    <span class="text-[8px] text-cyan-400 font-black">MBPS</span>
                                                                </div>
                                                            {/if}
                                                        {/if}
                                                    </div>
                                                {/if}
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
