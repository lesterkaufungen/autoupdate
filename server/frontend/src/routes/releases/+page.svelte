<script lang="ts">
	import { onMount } from 'svelte';
	import { slide } from 'svelte/transition';
	import { api, type Release, type App } from '$lib/api';
	import { Upload, Plus, X, ChevronLeft, ChevronRight, Search, Filter, Calendar, Clock } from 'lucide-svelte';

	let releases = $state<Release[]>([]);
	let apps = $state<App[]>([]);
	let loading = $state(true);
	let error = $state('');

	let showUploadModal = $state(false);
	let selectedApp = $state('');
	let releaseVersion = $state('');
	let isScheduled = $state(false);
	let scheduleDate = $state('');
	let scheduleTime = $state('');
	
	let binaries = $state<{ os: string; arch: string; file: File | null }[]>([
		{ os: 'linux', arch: 'amd64', file: null }
	]);
	let uploading = $state(false);
	let uploadProgress = $state(0);

	// Pagination & Filtering
	let page = $state(1);
	let limit = $state(10);
	let total = $state(0);
	let filterAppId = $state('');
	let filterVersion = $state('');

	async function loadReleases() {
		loading = true;
		try {
			const res = await api.releases.list({
				appId: filterAppId,
				version: filterVersion,
				page,
				limit
			});
			releases = res.items;
			total = res.total;
		} catch (e: any) {
			error = e.message;
		} finally {
			loading = false;
		}
	}

	async function loadApps() {
		try {
			apps = await api.apps.list();
			if (apps.length > 0 && !selectedApp) selectedApp = apps[0].id;
		} catch (e: any) {
			console.error('Failed to load apps:', e);
		}
	}

	onMount(async () => {
		await Promise.all([loadReleases(), loadApps()]);
	});

	// Effect-like behavior for filters and pagination
	$effect(() => {
		if (filterAppId || filterVersion || page) {
			loadReleases();
		}
	});

	function addBinary() {
		binaries = [...binaries, { os: 'linux', arch: 'amd64', file: null }];
	}

	function removeBinary(index: number) {
		binaries = binaries.filter((_, i) => i !== index);
	}

	async function handleUpload() {
		if (!selectedApp || !releaseVersion || binaries.every(b => !b.file)) return;
		uploading = true;
		uploadProgress = 0;
		try {
			const fd = new FormData();
			fd.append('app_id', selectedApp);
			fd.append('version', releaseVersion);
			
			if (isScheduled && scheduleDate && scheduleTime) {
				const date = new Date(`${scheduleDate}T${scheduleTime}`);
				fd.append('scheduled_at', date.toISOString());
			}
			
			binaries.forEach(b => {
				if (b.file) {
					fd.append(`binary_${b.os}_${b.arch}`, b.file);
				}
			});
			
			await api.releases.upload(fd, (p) => uploadProgress = p);
			showUploadModal = false;
			resetUploadForm();
			page = 1;
			await loadReleases();
		} catch (e: any) {
			alert(e.message);
		} finally {
			uploading = false;
			uploadProgress = 0;
		}
	}

	function resetUploadForm() {
		releaseVersion = '';
		isScheduled = false;
		scheduleDate = '';
		scheduleTime = '';
		binaries = [{ os: 'linux', arch: 'amd64', file: null }];
	}

	function formatDate(dateStr: string) {
		if (!dateStr) return 'N/A';
		const date = new Date(dateStr);
		if (isNaN(date.getTime())) return 'Invalid Date';
		return date.toLocaleString();
	}

	function formatSize(bytes: number) {
		const units = ['B', 'KB', 'MB', 'GB'];
		let size = bytes;
		let unitIndex = 0;
		while (size > 1024 && unitIndex < units.length - 1) {
			size /= 1024;
			unitIndex++;
		}
		return `${size.toFixed(2)} ${units[unitIndex]}`;
	}

	const totalPages = $derived(Math.ceil(total / limit));
</script>

<div class="space-y-8">
	<div class="flex justify-between items-end">
		<div>
			<h1 class="text-3xl font-bold">Releases</h1>
			<p class="text-zinc-400">Manage your binary versions and distribution.</p>
		</div>
		<button
			onclick={() => { resetUploadForm(); showUploadModal = true; }}
			class="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg font-medium flex items-center gap-2 transition-colors"
		>
			<Upload class="w-5 h-5" />
			Upload Release
		</button>
	</div>

	<!-- Filters -->
	<div class="grid grid-cols-1 md:grid-cols-4 gap-4 bg-zinc-900 p-4 rounded-xl border border-zinc-800">
		<div class="relative">
			<Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-zinc-500" />
			<input 
				type="text" 
				placeholder="Filter by version..." 
				bind:value={filterVersion}
				oninput={() => page = 1}
				class="w-full bg-zinc-950 border border-zinc-800 rounded-lg pl-10 pr-4 py-2 text-sm focus:outline-none focus:border-blue-500"
			/>
		</div>
		<div class="relative">
			<Filter class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-zinc-500" />
			<select 
				bind:value={filterAppId}
				onchange={() => page = 1}
				class="w-full bg-zinc-950 border border-zinc-800 rounded-lg pl-10 pr-4 py-2 text-sm focus:outline-none focus:border-blue-500 appearance-none"
			>
				<option value="">All Applications</option>
				{#each apps as app}
					<option value={app.id}>{app.name}</option>
				{/each}
			</select>
		</div>
	</div>

	{#if loading && releases.length === 0}
		<div class="space-y-4">
			{#each Array(5) as _}
				<div class="h-16 bg-zinc-900 rounded-lg border border-zinc-800 animate-pulse"></div>
			{/each}
		</div>
	{:else if error}
		<div class="p-4 bg-red-900/20 border border-red-900/50 rounded-lg text-red-500">
			{error}
		</div>
	{:else}
		<div class="bg-zinc-900 rounded-xl border border-zinc-800 overflow-hidden">
			<table class="w-full text-left">
				<thead>
					<tr class="bg-zinc-950 text-zinc-400 text-xs font-medium uppercase tracking-wider border-b border-zinc-800">
						<th class="px-6 py-4">Version</th>
						<th class="px-6 py-4">Application</th>
						<th class="px-6 py-4">Platform</th>
						<th class="px-6 py-4">Size</th>
						<th class="px-6 py-4">Created At</th>
						<th class="px-6 py-4">Scheduled</th>
						<th class="px-6 py-4">Status</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-zinc-800 text-sm">
					{#each (releases || []) as release}
						<tr class="hover:bg-zinc-800/50 transition-colors">
							<td class="px-6 py-4 font-bold text-zinc-100">v{release.version}</td>
							<td class="px-6 py-4 text-zinc-400">
								{(apps || []).find(a => a.id === release.app_id)?.name || release.app_id}
							</td>
							<td class="px-6 py-4">
								<span class="px-2 py-1 bg-zinc-800 rounded text-xs text-zinc-300">
									{release.os} / {release.arch}
								</span>
							</td>
							<td class="px-6 py-4 text-zinc-400">{formatSize(release.size)}</td>
							<td class="px-6 py-4 text-zinc-500">
								{formatDate(release.created_at)}
							</td>
							<td class="px-6 py-4 text-zinc-500">
								{release.scheduled_at ? formatDate(release.scheduled_at) : 'Immediate'}
							</td>
							<td class="px-6 py-4">
								{#if release.is_active}
									<span class="flex items-center gap-1.5 text-green-500">
										<span class="w-2 h-2 rounded-full bg-green-500"></span>
										Active
									</span>
								{:else}
									<span class="flex items-center gap-1.5 text-zinc-500">
										<span class="w-2 h-2 rounded-full bg-zinc-500"></span>
										Inactive
									</span>
								{/if}
							</td>
						</tr>
					{:else}
						<tr>
							<td colspan="7" class="px-6 py-12 text-center text-zinc-500">
								No releases found. Upload your first release to get started.
							</td>
						</tr>
					{/each}
				</tbody>
			</table>

			<!-- Pagination -->
			{#if totalPages > 1}
				<div class="px-6 py-4 bg-zinc-950 border-t border-zinc-800 flex items-center justify-between">
					<div class="text-sm text-zinc-500">
						Showing {(page - 1) * limit + 1} to {Math.min(page * limit, total)} of {total} releases
					</div>
					<div class="flex gap-2">
						<button 
							onclick={() => page = Math.max(1, page - 1)}
							disabled={page === 1}
							class="p-2 bg-zinc-900 border border-zinc-800 rounded-lg hover:bg-zinc-800 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
						>
							<ChevronLeft class="w-5 h-5" />
						</button>
						<div class="flex items-center px-4 text-sm font-medium">
							Page {page} of {totalPages}
						</div>
						<button 
							onclick={() => page = Math.min(totalPages, page + 1)}
							disabled={page === totalPages}
							class="p-2 bg-zinc-900 border border-zinc-800 rounded-lg hover:bg-zinc-800 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
						>
							<ChevronRight class="w-5 h-5" />
						</button>
					</div>
				</div>
			{/if}
		</div>
	{/if}
</div>

{#if showUploadModal}
	<div class="fixed inset-0 bg-black/80 backdrop-blur-sm flex items-center justify-center p-4 z-50">
		<div class="bg-zinc-900 border border-zinc-800 rounded-xl w-full max-w-2xl overflow-hidden shadow-2xl max-h-[90vh] flex flex-col">
			<div class="p-6 border-b border-zinc-800">
				<h2 class="text-xl font-bold">Upload New Release</h2>
			</div>
			<div class="p-6 space-y-6 overflow-y-auto flex-1">
				<div class="grid grid-cols-2 gap-4">
					<div class="space-y-1">
						<label for="app" class="text-sm font-medium text-zinc-400">Application</label>
						<select
							id="app"
							bind:value={selectedApp}
							class="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-4 py-2 focus:outline-none focus:border-blue-500"
						>
							{#each apps as app}
								<option value={app.id}>{app.name}</option>
							{/each}
						</select>
					</div>
					<div class="space-y-1">
						<label for="version" class="text-sm font-medium text-zinc-400">Release Version</label>
						<input
							id="version"
							type="text"
							bind:value={releaseVersion}
							class="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-4 py-2 focus:outline-none focus:border-blue-500"
							placeholder="e.g. 1.0.0"
						/>
					</div>
				</div>

				<div class="bg-zinc-950/50 p-4 rounded-xl border border-zinc-800 space-y-4">
					<div class="flex items-center justify-between">
						<div class="space-y-0.5">
							<label for="schedule-toggle" class="text-sm font-medium text-zinc-200">Schedule Release</label>
							<p class="text-xs text-zinc-500">Enable to set a future date and time for this release.</p>
						</div>
						<input
							id="schedule-toggle"
							type="checkbox"
							bind:checked={isScheduled}
							class="w-5 h-5 rounded border-zinc-800 bg-zinc-900 text-blue-600 focus:ring-blue-500 focus:ring-offset-zinc-900"
						/>
					</div>

					{#if isScheduled}
						<div transition:slide={{ duration: 200 }} class="grid grid-cols-2 gap-4 pt-4 border-t border-zinc-800/50">
							<div class="space-y-1">
								<label for="schedule_date" class="text-xs font-medium text-zinc-500 uppercase tracking-wider">Date</label>
								<div class="relative">
									<Calendar class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-zinc-500 pointer-events-none" />
									<input
										id="schedule_date"
										type="date"
										bind:value={scheduleDate}
										class="w-full bg-zinc-900 border border-zinc-800 rounded-lg pl-10 pr-4 py-2 text-sm focus:outline-none focus:border-blue-500 [color-scheme:dark]"
									/>
								</div>
							</div>
							<div class="space-y-1">
								<label for="schedule_time" class="text-xs font-medium text-zinc-500 uppercase tracking-wider">Time</label>
								<div class="relative">
									<Clock class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-zinc-500 pointer-events-none" />
									<input
										id="schedule_time"
										type="time"
										bind:value={scheduleTime}
										class="w-full bg-zinc-900 border border-zinc-800 rounded-lg pl-10 pr-4 py-2 text-sm focus:outline-none focus:border-blue-500 [color-scheme:dark]"
									/>
								</div>
							</div>
						</div>
					{/if}
				</div>

				<div class="space-y-4">
					<div class="flex justify-between items-center">
						<h3 class="text-sm font-bold text-zinc-500 uppercase tracking-widest">Binaries</h3>
						<button 
							onclick={addBinary}
							class="text-xs text-blue-500 hover:underline flex items-center gap-1"
						>
							<Plus class="w-3 h-3" /> Add Platform
						</button>
					</div>

					<div class="space-y-3">
						{#each binaries as binary, i}
							<div class="bg-zinc-950 p-4 rounded-xl border border-zinc-800 space-y-4 relative group/bin">
								<button 
									onclick={() => removeBinary(i)}
									class="absolute top-2 right-2 p-1 text-zinc-600 hover:text-red-500 transition-colors"
								>
									<X class="w-4 h-4" />
								</button>
								
								<div class="grid grid-cols-2 gap-4">
									<div class="space-y-1">
										<label class="text-xs text-zinc-500">OS</label>
										<select 
											bind:value={binary.os}
											class="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-1.5 text-sm focus:outline-none focus:border-blue-500"
										>
											<option value="linux">Linux</option>
											<option value="darwin">macOS</option>
											<option value="windows">Windows</option>
										</select>
									</div>
									<div class="space-y-1">
										<label class="text-xs text-zinc-500">Architecture</label>
										<select 
											bind:value={binary.arch}
											class="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-1.5 text-sm focus:outline-none focus:border-blue-500"
										>
											<option value="amd64">amd64</option>
											<option value="arm64">arm64</option>
											<option value="386">386</option>
										</select>
									</div>
								</div>

								<div class="relative h-20 border-2 border-dashed border-zinc-800 rounded-lg hover:border-zinc-700 transition-colors flex items-center justify-center p-4">
									<div class="text-center">
										<Upload class="w-6 h-6 text-zinc-600 mx-auto mb-1" />
										<p class="text-xs text-zinc-500">
											{binary.file ? binary.file.name : 'Select binary file'}
										</p>
									</div>
									<input 
										type="file" 
										class="absolute inset-0 opacity-0 cursor-pointer"
										onchange={(e) => binary.file = e.currentTarget.files?.[0] || null}
									/>
								</div>
							</div>
						{/each}
					</div>
				</div>
			</div>
			
			<div class="p-6 bg-zinc-950 flex flex-col gap-4 border-t border-zinc-800">
				{#if uploading}
					<div class="w-full space-y-2">
						<div class="flex justify-between text-xs text-zinc-400 font-medium">
							<span>Uploading binaries...</span>
							<span>{uploadProgress}%</span>
						</div>
						<div class="w-full h-2 bg-zinc-800 rounded-full overflow-hidden">
							<div 
								class="h-full bg-blue-600 transition-all duration-300" 
								style="width: {uploadProgress}%"
							></div>
						</div>
					</div>
				{/if}
				<div class="flex justify-end gap-3">
					<button
						onclick={() => showUploadModal = false}
						class="px-4 py-2 rounded-lg hover:bg-zinc-900 transition-colors"
					>
						Cancel
					</button>
					<button
						onclick={handleUpload}
						disabled={uploading || !releaseVersion || !selectedApp || binaries.every(b => !b.file)}
						class="bg-blue-600 hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed text-white px-6 py-2 rounded-lg font-medium transition-colors"
					>
						{uploading ? 'Uploading...' : 'Upload Release'}
					</button>
				</div>
			</div>
		</div>
	</div>
{/if}