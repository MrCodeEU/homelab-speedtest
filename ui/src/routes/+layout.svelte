<script>
	import '../app.css';
</script>

<div class="min-h-screen bg-gray-900 text-white font-sans selection:bg-cyan-500 selection:text-white">
    <nav class="border-b border-gray-800 bg-gray-900/50 backdrop-blur fixed top-0 w-full z-50">
        <div class="container mx-auto px-4 h-16 flex items-center justify-between">
            <div class="flex items-center space-x-8">
                <a href="/" class="text-xl font-bold bg-gradient-to-r from-cyan-400 to-blue-500 bg-clip-text text-transparent">
                    Homelab Speedtest
                </a>
                <div class="hidden md:flex space-x-6 text-sm font-medium">
                    <a href="/" class="hover:text-cyan-400 transition-colors">Dashboard</a>
                    <a href="/config" class="hover:text-cyan-400 transition-colors">Configuration</a>
                    <a href="/history" class="hover:text-cyan-400 transition-colors">History</a>
                </div>
            </div>
            <div>
                <a href="https://github.com/user/homelab-speedtest" target="_blank" class="text-gray-400 hover:text-white transition-colors">
                    GitHub
                </a>
            </div>
        </div>
    </nav>
    <main class="pt-24 px-4 pb-12 container mx-auto">
        <slot />
    </main>
</div>
