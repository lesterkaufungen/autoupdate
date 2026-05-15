<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { api, type Stats, type App, type Log } from '$lib/api';
	import { ws } from '$lib/ws';
	import { 
		BarChart3, 
		TrendingUp, 
		Users, 
		Globe, 
		Monitor, 
		Cpu, 
		ArrowUpRight,
		Info,
		ChevronRight,
		PieChart
	} from 'lucide-svelte';
	import { Chart, registerables } from 'chart.js';

	Chart.register(...registerables);

	let stats = $state<Stats[]>([]);
	let apps = $state<App[]>([]);
	let loading = $state(true);
	let error = $state('');

	let selectedAppId = $state(page.url.searchParams.get('app_id') || '');
	let dateRange = $state('7d'); // 7d, 30d, all

	let platformChartCanvas = $state<HTMLCanvasElement | null>(null);
	let versionChartCanvas = $state<HTMLCanvasElement | null>(null);
	let platformChart: Chart | null = null;
	let versionChart: Chart | null = null;

	async function loadStats() {
		if (!selectedAppId) {
			const a = await api.apps.list();
			apps = a;
			if (apps.length > 0) selectedAppId = apps[0].id;
			else {
				loading = false;
				return;
			}
		} else if (apps.length === 0) {
			apps = await api.apps.list();
		}

		loading = true;
		try {
			let from: string | undefined;
			if (dateRange === '7d') {
				from = new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString();
			} else if (dateRange === '30d') {
				from = new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString();
			}
			
			stats = await api.stats(selectedAppId, from);
			
			// Small delay to ensure canvases are ready
			setTimeout(initCharts, 0);
		} catch (e: any) {
			error = e.message;
		} finally {
			loading = false;
		}
	}

	function initCharts() {
		initPlatformChart();
		initVersionChart();
	}

	function initPlatformChart() {
		if (!platformChartCanvas || !globalDistribution.platforms) return;
		if (platformChart) platformChart.destroy();

		const isDark = document.documentElement.classList.contains('dark');
		const data = sortedPlatforms;
		const ctx = platformChartCanvas.getContext('2d');
		if (!ctx) return;

		platformChart = new Chart(ctx, {
			type: 'doughnut',
			data: {
				labels: data.map(d => d[0]),
				datasets: [{
					data: data.map(d => d[1].total),
					backgroundColor: [
						'#3b82f6', '#10b981', '#f59e0b', '#8b5cf6', '#ef4444', '#06b6d4'
					],
					borderWidth: 0,
					hoverOffset: 4
				}]
			},
			options: {
				responsive: true,
				maintainAspectRatio: false,
				plugins: {
					legend: {
						position: 'right',
						labels: {
							color: isDark ? '#71717a' : '#52525b',
							usePointStyle: true,
							padding: 20,
							font: { size: 11 }
						}
					},
					tooltip: {
						backgroundColor: isDark ? '#18181b' : '#ffffff',
						titleColor: isDark ? '#a1a1aa' : '#18181b',
						bodyColor: isDark ? '#fafafa' : '#18181b',
						borderColor: isDark ? '#27272a' : '#e4e4e7',
						borderWidth: 1,
						padding: 12,
						cornerRadius: 8
					}
				},
				cutout: '70%'
			}
		});
	}

	function initVersionChart() {
		if (!versionChartCanvas || stats.length === 0) return;
		if (versionChart) versionChart.destroy();

		const isDark = document.documentElement.classList.contains('dark');
		const data = sortedStats;
		const ctx = versionChartCanvas.getContext('2d');
		if (!ctx) return;

		versionChart = new Chart(ctx, {
			type: 'bar',
			data: {
				labels: data.map(s => `v${s.version}`),
				datasets: [
					{
						label: 'Active',
						data: data.map(s => s.active),
						backgroundColor: '#4f46e5',
						borderRadius: 4
					},
					{
						label: 'New',
						data: data.map(s => s.new),
						backgroundColor: '#10b981',
						borderRadius: 4
					}
				]
			},
			options: {
				responsive: true,
				maintainAspectRatio: false,
				scales: {
					x: {
						stacked: true,
						grid: { display: false },
						ticks: { color: isDark ? '#71717a' : '#52525b' }
					},
					y: {
						stacked: true,
						grid: { color: isDark ? 'rgba(39, 39, 42, 0.5)' : 'rgba(228, 228, 231, 0.5)' },
						ticks: { color: isDark ? '#71717a' : '#52525b' }
					}
				},
				plugins: {
					legend: {
						display: false
					},
					tooltip: {
						backgroundColor: isDark ? '#18181b' : '#ffffff',
						titleColor: isDark ? '#a1a1aa' : '#18181b',
						bodyColor: isDark ? '#fafafa' : '#18181b',
						borderColor: isDark ? '#27272a' : '#e4e4e7',
						borderWidth: 1,
						padding: 12,
						cornerRadius: 8
					}
				}
			}
		});
	}

	onMount(() => {
		loadStats();
		
		// Re-initialize charts on theme change
		const observer = new MutationObserver(() => {
			initCharts();
		});
		observer.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] });

		const unsubscribe = ws.subscribe((topic, data) => {
			if (topic === 'analytics' && (data as Log).app_id === selectedAppId) {
				api.stats(selectedAppId).then(res => {
					stats = res;
					initCharts();
				});
			}
		});

		return () => {
			observer.disconnect();
			unsubscribe();
		};
	});

	const totals = $derived((stats || []).reduce((acc, curr) => ({
		new: acc.new + curr.new,
		active: acc.active + curr.active,
		total: acc.total + curr.new + curr.active
	}), { new: 0, active: 0, total: 0 }));

	const globalDistribution = $derived((stats || []).reduce((acc, s) => {
		Object.entries(s.platforms || {}).forEach(([p, data]) => {
			if (!acc.platforms[p]) acc.platforms[p] = { new: 0, active: 0, total: 0 };
			acc.platforms[p].new += data.new;
			acc.platforms[p].active += data.active;
			acc.platforms[p].total += data.new + data.active;
		});
		Object.entries(s.countries || {}).forEach(([c, platforms]) => {
			const countryCode = c.toUpperCase();
			if (!acc.countries[countryCode]) acc.countries[countryCode] = { new: 0, active: 0, total: 0 };
			Object.values(platforms).forEach(data => {
				acc.countries[countryCode].new += data.new;
				acc.countries[countryCode].active += data.active;
				acc.countries[countryCode].total += data.new + data.active;
			});
		});
		return acc;
	}, { 
		platforms: {} as Record<string, { new: number; active: number, total: number }>, 
		countries: {} as Record<string, { new: number; active: number, total: number }> 
	}));

	const sortedCountries = $derived(Object.entries(globalDistribution.countries)
		.sort((a, b) => b[1].total - a[1].total));

	const sortedPlatforms = $derived(Object.entries(globalDistribution.platforms)
		.sort((a, b) => b[1].total - a[1].total));

	const retentionRate = $derived(totals.total > 0 ? (totals.active / totals.total * 100).toFixed(1) : '0');

	const sortedStats = $derived([...stats].sort((a, b) => 
		b.version.localeCompare(a.version, undefined, { numeric: true, sensitivity: 'base' })
	));
</script>

<div class="space-y-8 pb-12">
	<!-- Header -->
	<div class="flex flex-col md:flex-row md:items-center justify-between gap-4">
		<div>
			<h1 class="text-3xl font-bold tracking-tight">Analytics</h1>
			<p class="text-zinc-500 mt-1">Detailed insights into your application's reach and retention.</p>
		</div>
		<div class="flex items-center gap-3">
			<select
				bind:value={dateRange}
				onchange={loadStats}
				class="bg-zinc-900 border border-zinc-800 rounded-xl px-4 py-2 text-sm font-medium focus:outline-none focus:ring-2 focus:ring-blue-500/20"
			>
				<option value="7d">Last 7 Days</option>
				<option value="30d">Last 30 Days</option>
				<option value="all">All Time</option>
			</select>
			
			{#if apps.length > 0}
				<select
					bind:value={selectedAppId}
					onchange={loadStats}
					class="bg-zinc-900 border border-zinc-800 rounded-xl px-4 py-2 text-sm font-medium focus:outline-none focus:ring-2 focus:ring-blue-500/20"
				>
					{#each apps as app}
						<option value={app.id}>{app.name}</option>
					{/each}
				</select>
			{/if}
		</div>
	</div>

	{#if loading && (!stats || stats.length === 0)}
		<div class="grid grid-cols-1 md:grid-cols-4 gap-6 animate-pulse">
			{#each Array(4) as _}
				<div class="h-32 bg-zinc-900 rounded-2xl border border-zinc-800"></div>
			{/each}
		</div>
		<div class="grid grid-cols-1 lg:grid-cols-2 gap-8">
			<div class="h-96 bg-zinc-900 rounded-2xl border border-zinc-800 animate-pulse"></div>
			<div class="h-96 bg-zinc-900 rounded-2xl border border-zinc-800 animate-pulse"></div>
		</div>
	{:else if error}
		<div class="p-6 bg-red-900/20 border border-red-900/50 rounded-2xl text-red-500 flex items-center gap-3">
			<Info class="w-5 h-5" />
			<p>{error}</p>
		</div>
	{:else if !stats || stats.length === 0}
		<div class="p-20 text-center bg-zinc-900/50 rounded-3xl border border-zinc-800 border-dashed text-zinc-500">
			<div class="inline-flex p-4 rounded-2xl bg-zinc-900 mb-4">
				<BarChart3 class="w-10 h-10 text-zinc-700" />
			</div>
			<p class="text-xl font-semibold text-zinc-200">No Data Yet</p>
			<p class="mt-2 max-w-sm mx-auto">
				{#if apps.length === 0}
					Create an application to start tracking statistics.
				{:else}
					No interactions recorded for this application in the selected period.
				{/if}
			</p>
		</div>
	{:else}
		<!-- Stats Grid -->
		<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
			<div class="p-6 bg-zinc-900 rounded-2xl border border-zinc-800 transition-all hover:border-zinc-700">
				<div class="flex justify-between items-start mb-4">
					<div class="p-2 bg-blue-500/10 rounded-lg">
						<Users class="w-5 h-5 text-blue-500" />
					</div>
					<span class="text-[10px] font-bold text-zinc-600 uppercase tracking-widest">Reach</span>
				</div>
				<p class="text-4xl font-bold tracking-tight">{totals.total.toLocaleString()}</p>
				<p class="text-xs text-zinc-500 mt-2 font-medium">Total Unique Nodes</p>
			</div>

			<div class="p-6 bg-zinc-900 rounded-2xl border border-zinc-800 transition-all hover:border-zinc-700">
				<div class="flex justify-between items-start mb-4">
					<div class="p-2 bg-emerald-500/10 rounded-lg">
						<TrendingUp class="w-5 h-5 text-emerald-500" />
					</div>
					<span class="text-[10px] font-bold text-zinc-600 uppercase tracking-widest">Growth</span>
				</div>
				<div class="flex items-baseline gap-2">
					<p class="text-4xl font-bold tracking-tight text-emerald-500">+{totals.new.toLocaleString()}</p>
					{#if totals.total > 0}
						<span class="text-xs font-bold text-emerald-600">+{((totals.new / totals.total) * 100).toFixed(0)}%</span>
					{/if}
				</div>
				<p class="text-xs text-zinc-500 mt-2 font-medium">New Nodes</p>
			</div>

			<div class="p-6 bg-zinc-900 rounded-2xl border border-zinc-800 transition-all hover:border-zinc-700">
				<div class="flex justify-between items-start mb-4">
					<div class="p-2 bg-indigo-500/10 rounded-lg">
						<ArrowUpRight class="w-5 h-5 text-indigo-500" />
					</div>
					<span class="text-[10px] font-bold text-zinc-600 uppercase tracking-widest">Retention</span>
				</div>
				<div class="flex items-baseline gap-2">
					<p class="text-4xl font-bold tracking-tight text-indigo-500">{retentionRate}%</p>
				</div>
				<p class="text-xs text-zinc-500 mt-2 font-medium">Active User Rate</p>
			</div>

			<div class="p-6 bg-zinc-900 rounded-2xl border border-zinc-800 transition-all hover:border-zinc-700">
				<div class="flex justify-between items-start mb-4">
					<div class="p-2 bg-purple-500/10 rounded-lg">
						<Globe class="w-5 h-5 text-purple-500" />
					</div>
					<span class="text-[10px] font-bold text-zinc-600 uppercase tracking-widest">Global</span>
				</div>
				<p class="text-4xl font-bold tracking-tight">{Object.keys(globalDistribution.countries).length}</p>
				<p class="text-xs text-zinc-500 mt-2 font-medium">Countries Reached</p>
			</div>
		</div>

		<div class="grid grid-cols-1 lg:grid-cols-2 gap-8">
			<!-- Platform Distribution -->
			<div class="bg-zinc-900 rounded-2xl border border-zinc-800 overflow-hidden">
				<div class="p-6 border-b border-zinc-800 flex justify-between items-center">
					<h3 class="font-bold flex items-center gap-2 text-sm uppercase tracking-widest text-zinc-400">
						<PieChart class="w-4 h-4" />
						Platforms
					</h3>
				</div>
				<div class="p-6 h-64">
					<canvas bind:this={platformChartCanvas}></canvas>
				</div>
			</div>

			<!-- Version Adoption -->
			<div class="bg-zinc-900 rounded-2xl border border-zinc-800 overflow-hidden">
				<div class="p-6 border-b border-zinc-800 flex justify-between items-center">
					<h3 class="font-bold flex items-center gap-2 text-sm uppercase tracking-widest text-zinc-400">
						<BarChart3 class="w-4 h-4" />
						Version Adoption
					</h3>
					<div class="flex gap-4">
						<div class="flex items-center gap-2">
							<div class="w-2 h-2 rounded-full bg-indigo-600"></div>
							<span class="text-[10px] font-bold text-zinc-500 uppercase">Active</span>
						</div>
						<div class="flex items-center gap-2">
							<div class="w-2 h-2 rounded-full bg-emerald-500"></div>
							<span class="text-[10px] font-bold text-zinc-500 uppercase">New</span>
						</div>
					</div>
				</div>
				<div class="p-6 h-64">
					<canvas bind:this={versionChartCanvas}></canvas>
				</div>
			</div>
		</div>

		<div class="grid grid-cols-1 lg:grid-cols-3 gap-8">
			<!-- Geographic Distribution -->
			<div class="lg:col-span-2 space-y-6">
				<div class="bg-zinc-900 rounded-2xl border border-zinc-800 overflow-hidden">
					<div class="p-6 border-b border-zinc-800 flex justify-between items-center">
						<h3 class="font-bold flex items-center gap-2 text-sm uppercase tracking-widest text-zinc-400">
							<Globe class="w-4 h-4" />
							Geographic Distribution
						</h3>
						<span class="text-xs font-medium text-zinc-500">{sortedCountries.length} regions</span>
					</div>
					<div class="divide-y divide-zinc-800/50">
						<div class="grid grid-cols-1 sm:grid-cols-2">
							{#each sortedCountries as [code, data], i}
								<div class="p-4 flex items-center justify-between hover:bg-zinc-800/30 transition-colors {i % 2 === 0 ? 'sm:border-r border-zinc-800/50' : ''} border-b border-zinc-800/50">
									<div class="flex items-center gap-3">
										<span class="text-xs font-mono font-bold text-zinc-500 w-4">{i + 1}</span>
										{#if code !== 'UNKNOWN'}
											<img 
												src="https://flagcdn.com/{code.toLowerCase()}.svg" 
												alt={code}
												class="w-6 h-4 rounded-sm object-cover shadow-sm"
											/>
										{/if}
										<span class="font-bold text-sm tracking-tight">{code === 'UNKNOWN' ? 'Unknown' : code}</span>
									</div>
									<div class="flex items-center gap-6">
										<div class="text-right">
											<p class="text-sm font-bold">{data.total.toLocaleString()}</p>
											<p class="text-[10px] font-bold text-zinc-400 uppercase tracking-wider">Total</p>
										</div>
										<div class="text-right">
											<p class="text-sm font-bold text-emerald-500">{data.new.toLocaleString()}</p>
											<p class="text-[10px] font-bold text-emerald-500/70 uppercase tracking-wider">New</p>
										</div>
									</div>
								</div>
							{/each}
						</div>
					</div>
				</div>
			</div>

			<!-- Raw Version Stats Table -->
			<div class="bg-zinc-900 rounded-2xl border border-zinc-800 overflow-hidden self-start">
				<div class="p-6 border-b border-zinc-800">
					<h3 class="font-bold flex items-center gap-2 text-sm uppercase tracking-widest text-zinc-400">
						<Info class="w-4 h-4" />
						Version Breakdown
					</h3>
				</div>
				<div class="divide-y divide-zinc-800">
					{#each sortedStats as s}
						<div class="p-4 hover:bg-zinc-800/30 transition-colors">
							<div class="flex justify-between items-center mb-2">
								<span class="font-mono font-bold text-blue-500">v{s.version}</span>
								<span class="text-xs font-bold">{s.new + s.active} nodes</span>
							</div>
							<div class="w-full h-1.5 bg-zinc-950 rounded-full overflow-hidden flex">
								<div class="bg-emerald-500 h-full" style="width: {(s.new / (s.new + s.active)) * 100}%"></div>
								<div class="bg-indigo-600 h-full" style="width: {(s.active / (s.new + s.active)) * 100}%"></div>
							</div>
							<div class="flex justify-between mt-1 text-[10px] text-zinc-500 font-medium">
								<span>{((s.new / totals.total) * 100).toFixed(1)}% of total</span>
								<span>{s.new} new / {s.active} active</span>
							</div>
						</div>
					{/each}
				</div>
			</div>
		</div>
	{/if}
</div>
