<script lang="ts">
	import { onMount } from 'svelte';
	import { api, type App, type Release, type DashboardStats, type TrendPoint } from '$lib/api';
	import { 
		Box, 
		Upload, 
		Activity, 
		Users, 
		TrendingUp, 
		ArrowUpRight, 
		ArrowDownRight,
		Clock,
		ExternalLink,
		ChevronRight,
		Globe,
		Monitor,
		ShieldCheck
	} from 'lucide-svelte';
	import { Chart, registerables } from 'chart.js';

	Chart.register(...registerables);

	let apps = $state<App[]>([]);
	let releases = $state<Release[]>([]);
	let dashStats = $state<DashboardStats | null>(null);
	let trends = $state<TrendPoint[]>([]);
	let loading = $state(true);
	let error = $state('');

	let chartCanvas = $state<HTMLCanvasElement | null>(null);
	let chart: Chart | null = null;

	onMount(async () => {
		// Re-initialize charts on theme change
		const observer = new MutationObserver(() => {
			initChart();
		});
		observer.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] });

		try {
			const [appsRes, releasesRes, statsRes, trendsRes] = await Promise.all([
				api.apps.list(),
				api.releases.list({ limit: 10 }),
				api.dashboard.stats(),
				api.dashboard.trends({ days: 14 })
			]);
			apps = appsRes;
			releases = releasesRes.items;
			dashStats = statsRes;
			trends = trendsRes;
		} catch (e: any) {
			error = e.message;
		} finally {
			loading = false;
		}

		return () => {
			observer.disconnect();
		};
	});

	$effect(() => {
		if (!loading && chartCanvas && trends.length > 0) {
			initChart();
		}
	});

	function initChart() {
		if (!chartCanvas || trends.length === 0) return;

		const ctx = chartCanvas.getContext('2d');
		if (!ctx) return;

		if (chart) {
			chart.destroy();
		}

		chart = new Chart(ctx, {
			type: 'line',
			data: {
				labels: trends.map(t => new Date(t.date).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })),
				datasets: [
					{
						label: 'Checks',
						data: trends.map(t => t.checks),
						borderColor: '#3b82f6',
						backgroundColor: 'rgba(59, 130, 246, 0.1)',
						fill: true,
						tension: 0.4,
						borderWidth: 2,
						pointRadius: 0,
						pointHoverRadius: 4,
					},
					{
						label: 'Downloads',
						data: trends.map(t => t.downloads),
						borderColor: '#10b981',
						backgroundColor: 'rgba(16, 185, 129, 0.1)',
						fill: true,
						tension: 0.4,
						borderWidth: 2,
						pointRadius: 0,
						pointHoverRadius: 4,
					}
				]
			},
			options: {
				responsive: true,
				maintainAspectRatio: false,
				interaction: {
					intersect: false,
					mode: 'index',
				},
				plugins: {
					legend: {
						display: false,
					},
					tooltip: {
						backgroundColor: document.documentElement.classList.contains('dark') ? '#18181b' : '#ffffff',
						titleColor: document.documentElement.classList.contains('dark') ? '#a1a1aa' : '#18181b',
						bodyColor: document.documentElement.classList.contains('dark') ? '#fafafa' : '#18181b',
						borderColor: document.documentElement.classList.contains('dark') ? '#27272a' : '#e4e4e7',
						borderWidth: 1,
						padding: 12,
						cornerRadius: 8,
						displayColors: true,
					}
				},
				scales: {
					x: {
						grid: {
							display: false,
						},
						ticks: {
							color: document.documentElement.classList.contains('dark') ? '#71717a' : '#52525b',
							font: {
								size: 10
							}
						}
					},
					y: {
						beginAtZero: true,
						grid: {
							color: document.documentElement.classList.contains('dark') ? 'rgba(39, 39, 42, 0.5)' : 'rgba(228, 228, 231, 0.5)',
						},
						ticks: {
							color: document.documentElement.classList.contains('dark') ? '#71717a' : '#52525b',
							font: {
								size: 10
							},
							stepSize: 1
						}
					}
				}
			}
		});
	}

	function formatDate(dateStr: string) {
		if (!dateStr) return 'N/A';
		const date = new Date(dateStr);
		if (isNaN(date.getTime())) return 'Invalid Date';
		return date.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
	}

	function timeAgo(dateStr?: string) {
		if (!dateStr) return '';
		const date = new Date(dateStr);
		const now = new Date();
		const diff = now.getTime() - date.getTime();
		const seconds = Math.floor(diff / 1000);
		const minutes = Math.floor(seconds / 60);
		const hours = Math.floor(minutes / 60);
		const days = Math.floor(hours / 24);

		if (days > 0) return `${days}d ago`;
		if (hours > 0) return `${hours}h ago`;
		if (minutes > 0) return `${minutes}m ago`;
		return 'Just now';
	}

	const stats = $derived([
		{ 
			label: 'Applications', 
			value: apps?.length ?? 0, 
			icon: Box, 
			color: 'text-blue-500', 
			bg: 'bg-blue-500/10',
			trend: '+12%', 
			trendUp: true 
		},
		{ 
			label: 'Nodes Active', 
			value: dashStats?.total_nodes ?? 0, 
			icon: Users, 
			color: 'text-orange-500', 
			bg: 'bg-orange-500/10',
			trend: '+5.4%', 
			trendUp: true 
		},
		{ 
			label: 'Total Downloads', 
			value: dashStats?.total_downloads ?? 0, 
			icon: Activity, 
			color: 'text-purple-500', 
			bg: 'bg-purple-500/10',
			trend: '+18%', 
			trendUp: true 
		},
		{ 
			label: 'Recent Updates', 
			value: (releases || []).filter(r => r.is_active).length, 
			icon: ShieldCheck, 
			color: 'text-emerald-500', 
			bg: 'bg-emerald-500/10',
			trend: 'Stable', 
			trendUp: null 
		},
	]);
</script>

<div class="space-y-8 pb-12">
	<div class="flex justify-between items-center">
		<div>
			<h1 class="text-3xl font-bold tracking-tight">System Overview</h1>
			<p class="text-zinc-500 mt-1">Real-time health and distribution metrics for all managed applications.</p>
		</div>
		<div class="flex items-center gap-2 px-3 py-1.5 bg-zinc-900 border border-zinc-800 rounded-full">
			<div class="w-2 h-2 bg-emerald-500 rounded-full animate-pulse"></div>
			<span class="text-xs font-bold text-zinc-400 uppercase tracking-widest">System Online</span>
		</div>
	</div>

	{#if loading}
		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 animate-pulse">
			{#each Array(4) as _}
				<div class="h-32 bg-zinc-900 rounded-2xl border border-zinc-800"></div>
			{/each}
		</div>
		<div class="h-[400px] bg-zinc-900 rounded-2xl border border-zinc-800 animate-pulse"></div>
	{:else if error}
		<div class="p-6 bg-red-900/20 border border-red-900/50 rounded-2xl text-red-500 flex items-center gap-4">
			<Activity class="w-6 h-6" />
			<div>
				<p class="font-bold">Failed to load dashboard data</p>
				<p class="text-sm opacity-80">{error}</p>
			</div>
		</div>
	{:else}
		<!-- Stats Grid -->
		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
			{#each stats as stat}
				<div class="p-6 bg-zinc-900 rounded-2xl border border-zinc-800 group hover:border-zinc-700 transition-all">
					<div class="flex justify-between items-start mb-4">
						<div class="p-2.5 {stat.bg} rounded-xl">
							<stat.icon class="w-5 h-5 {stat.color}" />
						</div>
						{#if stat.trendUp !== null}
							<div class="flex items-center gap-1 {stat.trendUp ? 'text-emerald-500' : 'text-red-500'} text-xs font-bold">
								{stat.trend}
								{#if stat.trendUp}
									<ArrowUpRight class="w-3 h-3" />
								{:else}
									<ArrowDownRight class="w-3 h-3" />
								{/if}
							</div>
						{/if}
					</div>
					<p class="text-sm font-medium text-zinc-500">{stat.label}</p>
					<p class="text-3xl font-bold tracking-tight mt-1">{stat.value.toLocaleString()}</p>
				</div>
			{/each}
		</div>

		<!-- Main Chart & Recent Activity -->
		<div class="grid grid-cols-1 lg:grid-cols-3 gap-8">
			<div class="lg:col-span-2 space-y-8">
				<!-- Traffic Chart -->
				<div class="bg-zinc-900 rounded-2xl border border-zinc-800 overflow-hidden">
					<div class="p-6 border-b border-zinc-800 flex justify-between items-center">
						<div>
							<h3 class="font-bold flex items-center gap-2">
								<Activity class="w-4 h-4 text-blue-500" />
								Update Traffic
							</h3>
							<p class="text-xs text-zinc-500 mt-0.5">Application checks and successful downloads over time.</p>
						</div>
						<div class="flex gap-4">
							<div class="flex items-center gap-2">
								<div class="w-2 h-2 rounded-full bg-blue-500"></div>
								<span class="text-[10px] font-bold text-zinc-500 uppercase">Checks</span>
							</div>
							<div class="flex items-center gap-2">
								<div class="w-2 h-2 rounded-full bg-emerald-500"></div>
								<span class="text-[10px] font-bold text-zinc-500 uppercase">Downloads</span>
							</div>
						</div>
					</div>
					<div class="p-6 h-80">
						<canvas bind:this={chartCanvas}></canvas>
					</div>
				</div>

				<!-- Active Applications -->
				<div class="bg-zinc-900 rounded-2xl border border-zinc-800 overflow-hidden">
					<div class="p-6 border-b border-zinc-800 flex justify-between items-center">
						<h3 class="font-bold flex items-center gap-2">
							<Box class="w-4 h-4 text-zinc-400" />
							Active Applications
						</h3>
						<a href="/ui/apps" class="text-xs font-bold text-blue-500 hover:text-blue-400 flex items-center gap-1 transition-colors uppercase tracking-widest">
							All Apps <ChevronRight class="w-3 h-3" />
						</a>
					</div>
					<div class="overflow-x-auto">
						<table class="w-full text-left">
							<thead>
								<tr class="text-[10px] font-bold text-zinc-500 uppercase tracking-widest border-b border-zinc-800">
									<th class="px-6 py-4">Application</th>
									<th class="px-6 py-4">Status</th>
									<th class="px-6 py-4">Version</th>
									<th class="px-6 py-4 text-right">Reach</th>
								</tr>
							</thead>
							<tbody class="divide-y divide-zinc-800/50">
								{#each (apps || []).slice(0, 5) as app}
									<tr class="group hover:bg-zinc-800/30 transition-colors">
										<td class="px-6 py-4">
											<div class="flex items-center gap-3">
												<div class="w-8 h-8 rounded-lg bg-zinc-800 flex items-center justify-center overflow-hidden border border-zinc-700/50">
													{#if app.icon}
														<img src={app.icon} alt={app.name} class="w-full h-full object-cover" />
													{:else}
														<Box class="w-4 h-4 text-zinc-500" />
													{/if}
												</div>
												<div>
													<p class="font-bold text-sm">{app.name}</p>
													<p class="text-[10px] text-zinc-500 font-mono truncate max-w-[120px]">{app.id}</p>
												</div>
											</div>
										</td>
										<td class="px-6 py-4">
											<div class="flex items-center gap-1.5">
												<div class="w-1.5 h-1.5 rounded-full bg-emerald-500"></div>
												<span class="text-xs text-zinc-300">Active</span>
											</div>
										</td>
										<td class="px-6 py-4">
											<div class="flex flex-col">
												<span class="text-xs font-bold text-blue-500">v{app.latest_version || '0.0.0'}</span>
												<span class="text-[10px] text-zinc-500">{timeAgo(app.last_release_at) || 'No releases'}</span>
											</div>
										</td>
										<td class="px-6 py-4 text-right">
											<div class="flex flex-col items-end">
												<span class="text-sm font-bold">{app.total_installed?.toLocaleString() || 0}</span>
												<span class="text-[10px] text-zinc-500 uppercase tracking-tighter">Nodes</span>
											</div>
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				</div>
			</div>

			<!-- Sidebar -->
			<div class="space-y-8">
				<!-- Recent Activity -->
				<div class="bg-zinc-900 rounded-2xl border border-zinc-800 overflow-hidden">
					<div class="p-6 border-b border-zinc-800">
						<h3 class="font-bold flex items-center gap-2">
							<Clock class="w-4 h-4 text-zinc-400" />
							Recent Releases
						</h3>
					</div>
					<div class="divide-y divide-zinc-800">
						{#each (releases || []).slice(0, 8) as release}
							<div class="p-4 hover:bg-zinc-800/30 transition-colors group">
								<div class="flex justify-between items-start mb-1">
									<div class="flex items-center gap-2">
										<span class="text-sm font-bold text-zinc-100">v{release.version}</span>
										<span class="text-[10px] px-1.5 py-0.5 bg-zinc-800 rounded text-zinc-400 font-mono uppercase tracking-tighter">
											{release.os}
										</span>
									</div>
									<span class="text-[10px] text-zinc-500">{timeAgo(release.created_at)}</span>
								</div>
								<div class="flex justify-between items-center">
									<p class="text-[10px] text-zinc-500 truncate max-w-[150px]">
										{apps.find(a => a.id === release.app_id)?.name || 'Unknown App'}
									</p>
									<a href="/ui/releases?app_id={release.app_id}" class="opacity-0 group-hover:opacity-100 text-blue-500 transition-all">
										<ExternalLink class="w-3 h-3" />
									</a>
								</div>
							</div>
						{:else}
							<div class="p-8 text-center text-zinc-500 text-sm">No recent activity.</div>
						{/each}
					</div>
				</div>

				<!-- Geographic Info -->
				<div class="bg-zinc-900 rounded-2xl border border-zinc-800 p-6 space-y-6">
					<h3 class="font-bold flex items-center gap-2">
						<Globe class="w-4 h-4 text-zinc-400" />
						Global Adoption
					</h3>
					
					<div class="space-y-4">
						<div class="flex items-center justify-between text-xs">
							<span class="text-zinc-500">Highest Engagement</span>
							<span class="font-bold">United States</span>
						</div>
						<div class="flex items-center justify-between text-xs">
							<span class="text-zinc-500">Active Timezone</span>
							<span class="font-bold">UTC-5</span>
						</div>
						<div class="flex items-center justify-between text-xs">
							<span class="text-zinc-500">Main Platform</span>
							<span class="font-bold">Linux x64</span>
						</div>
					</div>

					<div class="pt-4 border-t border-zinc-800">
						<a href="/ui/stats" class="w-full flex items-center justify-center gap-2 py-2.5 bg-zinc-950 border border-zinc-800 rounded-xl text-xs font-bold text-zinc-400 hover:text-zinc-100 hover:border-zinc-700 transition-all uppercase tracking-widest">
							Detailed Analytics <ExternalLink class="w-3 h-3" />
						</a>
					</div>
				</div>
			</div>
		</div>
	{/if}
</div>
