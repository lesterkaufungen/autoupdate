<script lang="ts">
	import { onMount } from 'svelte';
	import { api, type Log, type App } from '$lib/api';
	import { ws } from '$lib/ws';
	import { List, Search, RefreshCw, ChevronLeft, ChevronRight, Monitor, Cpu, History, ArrowUpDown, ArrowUp, ArrowDown, Filter } from 'lucide-svelte';

	let logs = $state<Log[]>([]);
	let apps = $state<App[]>([]);
	let total = $state(0);
	let page = $state(1);
	let limit = $state(10);
	let loading = $state(true);
	let error = $state('');

	// Filters & Sorting
	let selectedAppId = $state('');
	let selectedAction = $state('');
	let search = $state('');
	let sortBy = $state('created_at');
	let order = $state<'asc' | 'desc'>('desc');

	let searchTimeout: any;

	async function loadLogs(p = page) {
		loading = true;
		try {
			const res = await api.logs({
				appId: selectedAppId || undefined,
				action: selectedAction || undefined,
				search: search || undefined,
				sortBy,
				order,
				page: p,
				limit
			});
			logs = res.items;
			total = res.total;
			page = res.page;
		} catch (e: any) {
			error = e.message;
		} finally {
			loading = false;
		}
	}

	async function loadApps() {
		try {
			apps = await api.apps.list();
		} catch (e) {}
	}

	function handleSearch() {
		clearTimeout(searchTimeout);
		searchTimeout = setTimeout(() => {
			loadLogs(1);
		}, 300);
	}

	function toggleSort(col: string) {
		if (sortBy === col) {
			order = order === 'asc' ? 'desc' : 'asc';
		} else {
			sortBy = col;
			order = 'desc';
		}
		loadLogs(1);
	}

	function formatDate(dateStr: string) {
		if (!dateStr) return 'N/A';
		const date = new Date(dateStr);
		if (isNaN(date.getTime())) return 'Invalid Date';
		return date.toLocaleString();
	}

	function timeAgo(dateStr: string) {
		if (!dateStr) return 'N/A';
		const date = new Date(dateStr);
		const now = new Date();
		const seconds = Math.floor((now.getTime() - date.getTime()) / 1000);
		
		if (seconds < 60) return 'just now';
		const minutes = Math.floor(seconds / 60);
		if (minutes < 60) return `${minutes}m ago`;
		const hours = Math.floor(minutes / 60);
		if (hours < 24) return `${hours}h ago`;
		const days = Math.floor(hours / 24);
		return `${days}d ago`;
	}

	onMount(() => {
		loadApps();
		loadLogs();
		return ws.subscribe((topic, data) => {
			if (topic === 'analytics') {
				const log = data as Log;
				// Only prepend if it matches current filters
				let matches = true;
				if (selectedAppId && log.app_id !== selectedAppId) matches = false;
				if (selectedAction && log.action !== selectedAction) matches = false;
				if (search && !log.node_id.includes(search) && !log.ip.includes(search)) matches = false;

				if (matches) {
					total++;
					// Only prepend to visible list if we are on the first page and sorted by newest first
					if (page === 1 && order === 'desc' && sortBy === 'created_at') {
						logs = [log, ...logs].slice(0, limit);
					}
				}
			}
		});
	});

	const totalPages = $derived(Math.ceil(total / limit));
</script>

<div class="space-y-6">
	<div class="flex flex-col md:flex-row md:items-center justify-between gap-4">
		<div>
			<h1 class="text-3xl font-bold">Activity Logs</h1>
			<p class="text-zinc-400">Real-time interaction logs from client nodes.</p>
		</div>
		<button
			onclick={() => loadLogs()}
			class="bg-zinc-900 hover:bg-zinc-800 text-zinc-100 px-4 py-2 rounded-lg font-medium flex items-center gap-2 border border-zinc-800 transition-colors self-start md:self-auto"
		>
			<RefreshCw class="w-4 h-4 {loading ? 'animate-spin' : ''}" />
			Refresh
		</button>
	</div>

	<!-- Filters Bar -->
	<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-3 bg-zinc-900/50 p-4 rounded-xl border border-zinc-800">
		<div class="relative lg:col-span-2">
			<Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-zinc-500" />
			<input
				type="text"
				placeholder="Search Node ID, IP..."
				bind:value={search}
				oninput={handleSearch}
				class="w-full bg-zinc-950 border border-zinc-800 rounded-lg pl-9 pr-4 py-2 text-sm focus:outline-none focus:border-blue-500 transition-colors"
			/>
		</div>

		<select
			bind:value={selectedAppId}
			onchange={() => loadLogs(1)}
			class="bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500 transition-colors"
		>
			<option value="">All Applications</option>
			{#each apps as app}
				<option value={app.id}>{app.name}</option>
			{/each}
		</select>

		<select
			bind:value={selectedAction}
			onchange={() => loadLogs(1)}
			class="bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500 transition-colors"
		>
			<option value="">All Actions</option>
			<option value="check">Check</option>
			<option value="download">Download</option>
		</select>

		<div class="flex items-center gap-2 text-xs text-zinc-500 justify-end px-2 lg:col-span-4">
			<Filter class="w-3 h-3" />
			<span>{total} logs found</span>
		</div>
	</div>

	{#if error}
		<div class="p-4 bg-red-900/20 border border-red-900/50 rounded-lg text-red-500">
			{error}
		</div>
	{:else}
		<div class="bg-zinc-900 rounded-xl border border-zinc-800 overflow-hidden">
			<div class="overflow-x-auto">
				<table class="w-full text-left">
					<thead>
						<tr class="bg-zinc-950 text-zinc-400 text-[10px] font-bold uppercase tracking-wider border-b border-zinc-800">
							<th class="px-6 py-4">
								<button onclick={() => toggleSort('created_at')} class="flex items-center gap-1 hover:text-zinc-200 transition-colors">
									Timestamp
									{#if sortBy === 'created_at'}
										{order === 'asc' ? '↑' : '↓'}
									{:else}
										<ArrowUpDown class="w-3 h-3 opacity-30" />
									{/if}
								</button>
							</th>
							<th class="px-6 py-4">Node Metadata</th>
							<th class="px-6 py-4">Action</th>
							<th class="px-6 py-4">Location & IP</th>
							<th class="px-6 py-4">
								<button onclick={() => toggleSort('version')} class="flex items-center gap-1 hover:text-zinc-200 transition-colors">
									Version
									{#if sortBy === 'version'}
										{order === 'asc' ? '↑' : '↓'}
									{:else}
										<ArrowUpDown class="w-3 h-3 opacity-30" />
									{/if}
								</button>
							</th>
							<th class="px-6 py-4">Platform</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-zinc-800 text-sm">
						{#each logs || [] as log}
							<tr class="hover:bg-zinc-800/50 transition-colors">
								<td class="px-6 py-4 text-zinc-400 whitespace-nowrap">
									<div class="flex flex-col">
										<span>{formatDate(log.created_at)}</span>
										<span class="text-[10px] text-zinc-600">{timeAgo(log.created_at)}</span>
									</div>
								</td>
								<td class="px-6 py-4">
									<div class="flex flex-col gap-1">
										<div class="flex items-center gap-2">
											<span class="font-mono text-xs text-blue-400 bg-blue-900/20 px-1.5 py-0.5 rounded">
												{log.node_id.slice(0, 8)}
											</span>
										</div>
										{#if log.node}
											<div class="flex items-center gap-3 text-[10px] text-zinc-500">
												<span class="flex items-center gap-1">
													<History class="w-3 h-3" />
													First seen {timeAgo(log.node.first_seen)}
												</span>
											</div>
										{/if}
									</div>
								</td>
								<td class="px-6 py-4">
									<span class="px-2 py-0.5 rounded text-[10px] font-bold uppercase {log.action === 'download' ? 'bg-indigo-900/40 text-indigo-400' : 'bg-zinc-800 text-zinc-400'}">
										{log.action}
									</span>
								</td>
								<td class="px-6 py-4">
									<div class="flex items-center gap-3">
										{#if log.country && log.country !== '??' && log.country !== 'UNKNOWN'}
											<img 
												src="https://flagcdn.com/{log.country.toLowerCase()}.svg" 
												alt={log.country}
												title={log.country}
												class="w-6 h-4 rounded-sm object-cover shadow-sm"
											/>
										{:else}
											<span class="text-lg leading-none" title="Unknown">🌍</span>
										{/if}
										<span class="text-[11px] text-zinc-500 font-mono">{log.ip}</span>
									</div>
								</td>
								<td class="px-6 py-4 text-zinc-300 font-mono text-xs">v{log.version}</td>
								<td class="px-6 py-4">
									<div class="flex items-center gap-2 text-zinc-500">
										<span class="flex items-center gap-1">
											<Monitor class="w-3 h-3" />
											{log.os}
										</span>
										<span class="text-zinc-700">/</span>
										<span class="flex items-center gap-1">
											<Cpu class="w-3 h-3" />
											{log.arch}
										</span>
									</div>
								</td>
							</tr>
						{:else}
							<tr>
								<td colspan="6" class="px-6 py-20 text-center text-zinc-500">
									{#if loading}
										<div class="flex flex-col items-center gap-2">
											<RefreshCw class="w-8 h-8 animate-spin text-zinc-700" />
											<span>Loading logs...</span>
										</div>
									{:else}
										<List class="w-12 h-12 mx-auto mb-4 text-zinc-800" />
										<p>No activity found matching your filters.</p>
										{#if selectedAppId || selectedAction || search}
											<button 
												onclick={() => { selectedAppId = ''; selectedAction = ''; search = ''; loadLogs(1); }}
												class="text-blue-500 text-sm mt-2 hover:underline"
											>
												Clear all filters
											</button>
										{/if}
									{/if}
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>

			{#if totalPages > 1}
				<div class="px-6 py-4 bg-zinc-950/50 border-t border-zinc-800 flex items-center justify-between">
					<div class="text-xs text-zinc-500">
						Showing <span class="text-zinc-300">{(page - 1) * limit + 1}</span> to <span class="text-zinc-300">{Math.min(page * limit, total)}</span> of <span class="text-zinc-300">{total}</span> results
					</div>
					<div class="flex items-center gap-2">
						<button
							disabled={page === 1 || loading}
							onclick={() => loadLogs(page - 1)}
							class="p-2 rounded-lg hover:bg-zinc-800 disabled:opacity-50 disabled:cursor-not-allowed transition-colors text-zinc-400"
						>
							<ChevronLeft class="w-4 h-4" />
						</button>
						<div class="flex items-center gap-1">
							{#each Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
								if (totalPages <= 5) return i + 1;
								if (page <= 3) return i + 1;
								if (page >= totalPages - 2) return totalPages - 4 + i;
								return page - 2 + i;
							}) as p}
								<button
									onclick={() => loadLogs(p)}
									class="w-8 h-8 rounded-lg text-xs font-medium transition-colors {page === p ? 'bg-blue-600 text-white' : 'text-zinc-400 hover:bg-zinc-800'}"
								>
									{p}
								</button>
							{/each}
						</div>
						<button
							disabled={page === totalPages || loading}
							onclick={() => loadLogs(page + 1)}
							class="p-2 rounded-lg hover:bg-zinc-800 disabled:opacity-50 disabled:cursor-not-allowed transition-colors text-zinc-400"
						>
							<ChevronRight class="w-4 h-4" />
						</button>
					</div>
				</div>
			{/if}
		</div>
	{/if}
</div>
