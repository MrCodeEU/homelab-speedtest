<script>
    import { onMount } from 'svelte';
    import { getSchedules, updateSchedule } from '$lib/api';

    /** @type {import('$lib/api').Schedule[]} */
    let schedules = [];
    let loading = true;
    /** @type {string|null} */
    let error = null;
    let saving = false;

    async function load() {
        try {
            schedules = await getSchedules();
        } catch (e) {
            error = e instanceof Error ? e.message : String(e);
        } finally {
            loading = false;
        }
    }

    /**
     * @param {import('$lib/api').Schedule} schedule
     */
    async function save(schedule) {
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

    onMount(load);
</script>

<div class="space-y-8">
    <header>
        <h1 class="text-3xl font-bold text-white">Configuration</h1>
        <p class="text-gray-400 mt-2">Manage test schedules and system settings.</p>
    </header>

    {#if loading}
        <div class="text-gray-400">Loading settings...</div>
    {:else if error}
        <div class="text-red-400">Error: {error}</div>
    {:else}
        <div class="grid gap-6 md:grid-cols-2">
            {#each schedules as schedule}
                <div class="bg-gray-800/50 border border-gray-700 rounded-xl p-6 backdrop-blur">
                    <div class="flex justify-between items-center mb-4">
                        <h2 class="text-xl font-semibold capitalize text-white">{schedule.type} Test</h2>
                        <span class={`px-2 py-1 rounded text-xs font-mono ${schedule.enabled ? 'bg-green-900 text-green-200' : 'bg-gray-700 text-gray-400'}`}>
                            {schedule.enabled ? 'ENABLED' : 'DISABLED'}
                        </span>
                    </div>

                    <div class="space-y-4">
                        <div>
                            <label class="block text-sm text-gray-400 mb-1" for="cron-{schedule.id}">Interval / Cron</label>
                            <input 
                                id="cron-{schedule.id}"
                                type="text" 
                                bind:value={schedule.cron} 
                                class="w-full bg-gray-900 border border-gray-700 rounded px-3 py-2 text-white font-mono focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none transition-colors"
                                placeholder="e.g. 1m, 30s"
                            />
                            <p class="text-xs text-gray-500 mt-1">Accepts Go durations (e.g. '30s', '5m')</p>
                        </div>

                        <div class="flex items-center">
                            <input 
                                id="enabled-{schedule.id}"
                                type="checkbox" 
                                bind:checked={schedule.enabled}
                                class="w-4 h-4 text-cyan-600 bg-gray-900 border-gray-700 rounded focus:ring-cyan-500 focus:ring-2"
                            />
                            <label for="enabled-{schedule.id}" class="ml-2 text-sm text-gray-300">Enable automated testing</label>
                        </div>

                        <button 
                            onclick={() => save(schedule)} 
                            disabled={saving}
                            class="w-full bg-cyan-600 hover:bg-cyan-500 text-white font-medium py-2 px-4 rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                        >
                            {saving ? 'Saving...' : 'Save Changes'}
                        </button>
                    </div>
                </div>
            {/each}
        </div>
    {/if}
</div>
