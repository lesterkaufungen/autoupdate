<script lang="ts">
	import { onMount } from 'svelte';
	import { slide } from 'svelte/transition';
	import { api, type App } from '$lib/api';
	import { Plus, Box, ExternalLink, Trash2, Edit2, Upload, X, Tag, Layers, TrendingUp, Users, Image as ImageIcon, Calendar, Clock } from 'lucide-svelte';

	let apps = $state<App[]>([]);
	let loading = $state(true);
	let error = $state('');

	let showCreateModal = $state(false);
	let showEditModal = $state(false);
	let showUploadModal = $state(false);

	let editingApp = $state<App | null>(null);
	let uploadingApp = $state<App | null>(null);

	let newAppName = $state('');
	let newAppDesc = $state('');
	let newAppIcon = $state('');
	let creating = $state(false);

	let editAppName = $state('');
	let editAppDesc = $state('');
	let editAppIcon = $state('');
	let updating = $state(false);

	// Upload state
	let releaseVersion = $state('');
	let isScheduled = $state(false);
	let scheduleDate = $state('');
	let scheduleTime = $state('');
	let binaries = $state<{ os: string; arch: string; file: File | null }[]>([
		{ os: 'linux', arch: 'amd64', file: null }
	]);
	let uploading = $state(false);
	let uploadProgress = $state(0);

	async function loadApps() {
		try {
			apps = await api.apps.list();
		} catch (e: any) {
			error = e.message;
		} finally {
			loading = false;
		}
	}

	function formatDate(dateStr: string) {
		if (!dateStr) return 'N/A';
		const date = new Date(dateStr);
		if (isNaN(date.getTime())) return 'Invalid Date';
		return date.toLocaleDateString();
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

	onMount(loadApps);

	async function createApp() {
		if (!newAppName) return;
		creating = true;
		try {
			await api.apps.create(newAppName, newAppDesc, newAppIcon);
			newAppName = '';
			newAppDesc = '';
			newAppIcon = '';
			showCreateModal = false;
			await loadApps();
		} catch (e: any) {
			alert(e.message);
		} finally {
			creating = false;
		}
	}

	async function updateApp() {
		if (!editingApp || !editAppName) return;
		updating = true;
		try {
			await api.apps.update(editingApp.id, editAppName, editAppDesc, editAppIcon);
			showEditModal = false;
			editingApp = null;
			await loadApps();
		} catch (e: any) {
			alert(e.message);
		} finally {
			updating = false;
		}
	}

	function openEdit(app: App) {
		editingApp = app;
		editAppName = app.name;
		editAppDesc = app.description;
		editAppIcon = app.icon || '';
		showEditModal = true;
	}

	function openUpload(app: App) {
		uploadingApp = app;
		releaseVersion = app.latest_version && app.latest_version !== 'N/A' ? app.latest_version : '1.0.0';
		resetUploadForm();
		showUploadModal = true;
	}

	function resetUploadForm() {
		isScheduled = false;
		scheduleDate = '';
		scheduleTime = '';
		binaries = [{ os: 'linux', arch: 'amd64', file: null }];
	}

	function addBinary() {
		binaries = [...binaries, { os: 'linux', arch: 'amd64', file: null }];
	}

	function removeBinary(index: number) {
		binaries = binaries.filter((_, i) => i !== index);
	}

	async function handleUpload() {
		if (!uploadingApp || !releaseVersion || binaries.every(b => !b.file)) return;
		uploading = true;
		uploadProgress = 0;
		try {
			const fd = new FormData();
			fd.append('app_id', uploadingApp.id);
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
			uploadingApp = null;
			resetUploadForm();
			await loadApps();
		} catch (e: any) {
			alert(e.message);
		} finally {
			uploading = false;
			uploadProgress = 0;
		}
	}
</script>

<div class="space-y-8">
	<div class="flex justify-between items-end">
		<div>
			<h1 class="text-3xl font-bold">Applications</h1>
			<p class="text-zinc-400">Manage your software products and releases.</p>
		</div>
		<button
			onclick={() => showCreateModal = true}
			class="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg font-medium flex items-center gap-2 transition-colors"
		>
			<Plus class="w-5 h-5" />
			Create App
		</button>
	</div>

	{#if loading}
		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
			{#each Array(3) as _}
				<div class="h-64 bg-zinc-900 rounded-xl border border-zinc-800 animate-pulse"></div>
			{/each}
		</div>
	{:else if error}
		<div class="p-4 bg-red-900/20 border border-red-900/50 rounded-lg text-red-500">
			{error}
		</div>
	{:else}
		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
			{#each apps || [] as app}
				<div class="bg-zinc-900 rounded-xl border border-zinc-800 p-5 flex flex-col justify-between group transition-all hover:border-zinc-700">
					<div>
						<div class="flex justify-between items-start mb-4">
							<div class="w-12 h-12 bg-zinc-800 rounded-xl group-hover:bg-blue-900/20 transition-colors flex items-center justify-center overflow-hidden border border-zinc-700/50">
								{#if app.icon}
									<img src={app.icon} alt={app.name} class="w-full h-full object-cover" />
								{:else}
									<Box class="w-6 h-6 text-blue-400" />
								{/if}
							</div>
							<div class="flex gap-1">
								<button 
									onclick={() => openEdit(app)}
									class="p-2 text-zinc-500 hover:text-zinc-100 hover:bg-zinc-800 rounded-lg transition-all"
								>
									<Edit2 class="w-4 h-4" />
								</button>
								<button 
									onclick={() => openUpload(app)}
									class="p-2 text-zinc-500 hover:text-blue-400 hover:bg-blue-900/20 rounded-lg transition-all"
								>
									<Upload class="w-4 h-4" />
								</button>
								<button class="p-2 text-zinc-500 hover:text-red-500 hover:bg-red-900/20 rounded-lg transition-all">
									<Trash2 class="w-4 h-4" />
								</button>
							</div>
						</div>
						<h3 class="text-xl font-bold mb-1">{app.name}</h3>
						<p class="text-sm text-zinc-400 mb-5 line-clamp-2 min-h-[2.5rem]">{app.description || 'No description provided.'}</p>
					</div>
					
					<div class="space-y-4">
						<div class="bg-zinc-950/50 rounded-lg border border-zinc-800/50 overflow-hidden">
							<div class="grid grid-cols-2 divide-x divide-y divide-zinc-800/30">
								<div class="p-2.5 flex flex-col">
									<span class="text-[9px] font-bold text-zinc-500 uppercase tracking-wider mb-0.5">Version</span>
									<div class="flex items-center gap-1.5">
										<span class="font-bold text-sm text-zinc-200">{app.latest_version || 'N/A'}</span>
										{#if app.last_release_at}
											<span class="text-[9px] text-zinc-500">{timeAgo(app.last_release_at)}</span>
										{/if}
									</div>
								</div>
								<div class="p-2.5 flex flex-col">
									<span class="text-[9px] font-bold text-zinc-500 uppercase tracking-wider mb-0.5">Releases</span>
									<span class="font-bold text-sm text-zinc-200">{app.release_count || 0}</span>
								</div>
								<div class="p-2.5 flex flex-col">
									<span class="text-[9px] font-bold text-zinc-500 uppercase tracking-wider mb-0.5">Today</span>
									<span class="font-bold text-sm text-zinc-200">+{app.today_installed || 0}</span>
								</div>
								<div class="p-2.5 flex flex-col">
									<span class="text-[9px] font-bold text-zinc-500 uppercase tracking-wider mb-0.5">Total Installs</span>
									<span class="font-bold text-sm text-zinc-200">{app.total_installed || 0}</span>
								</div>
							</div>
						</div>

						<div class="bg-zinc-950 rounded-lg p-2.5 text-[10px] font-mono break-all border border-zinc-800/50 flex justify-between items-center group/id">
							<span class="text-zinc-500 truncate mr-2">{app.id}</span>
							<button 
								class="text-zinc-600 hover:text-blue-500 opacity-0 group-hover/id:opacity-100 transition-all"
								onclick={() => navigator.clipboard.writeText(app.id)}
							>
								Copy
							</button>
						</div>
						
						<div class="flex justify-between items-center text-[10px] text-zinc-500">
							<span>Created {formatDate(app.created_at)}</span>
							<a href="/ui/stats?app_id={app.id}" class="text-blue-500 hover:text-blue-400 font-medium flex items-center gap-1 transition-colors uppercase tracking-widest text-[9px]">
								Analytics <ExternalLink class="w-3 h-3" />
							</a>
						</div>
					</div>
				</div>
			{:else}
				<div class="col-span-full py-20 text-center bg-zinc-900/50 rounded-xl border border-zinc-800 border-dashed">
					<Box class="w-12 h-12 text-zinc-700 mx-auto mb-4" />
					<h3 class="text-lg font-medium text-zinc-400">No applications yet</h3>
					<p class="text-sm text-zinc-500 mb-6">Create your first application to start managing updates.</p>
					<button
						onclick={() => showCreateModal = true}
						class="bg-blue-600 hover:bg-blue-700 text-white px-6 py-2 rounded-lg font-medium transition-colors"
					>
						Create an Application
					</button>
				</div>
			{/each}
		</div>
	{/if}
</div>

{#if showCreateModal}
	<div class="fixed inset-0 bg-black/80 backdrop-blur-sm flex items-center justify-center p-4 z-50">
		<div class="bg-zinc-900 border border-zinc-800 rounded-xl w-full max-w-md overflow-hidden shadow-2xl">
			<div class="p-6 border-b border-zinc-800">
				<h2 class="text-xl font-bold">New Application</h2>
			</div>
			<div class="p-6 space-y-4">
				<div class="space-y-1">
					<label for="name" class="text-sm font-medium text-zinc-400">App Name</label>
					<input
						id="name"
						type="text"
						bind:value={newAppName}
						class="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-4 py-2 focus:outline-none focus:border-blue-500"
						placeholder="e.g. My Awesome CLI"
					/>
				</div>
				<div class="space-y-1">
					<label for="desc" class="text-sm font-medium text-zinc-400">Description</label>
					<textarea
						id="desc"
						bind:value={newAppDesc}
						class="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-4 py-2 focus:outline-none focus:border-blue-500 h-24 resize-none"
						placeholder="What is this app for?"
					></textarea>
				</div>
				<div class="space-y-1">
					<label for="icon" class="text-sm font-medium text-zinc-400">Icon URL</label>
					<div class="flex gap-2">
						<div class="w-10 h-10 bg-zinc-950 border border-zinc-800 rounded-lg flex items-center justify-center shrink-0">
							{#if newAppIcon}
								<img src={newAppIcon} alt="Preview" class="w-full h-full object-cover rounded-lg" />
							{:else}
								<ImageIcon class="w-5 h-5 text-zinc-600" />
							{/if}
						</div>
						<input
							id="icon"
							type="text"
							bind:value={newAppIcon}
							class="flex-1 bg-zinc-950 border border-zinc-800 rounded-lg px-4 py-2 focus:outline-none focus:border-blue-500"
							placeholder="https://example.com/icon.png"
						/>
					</div>
				</div>
			</div>
			<div class="p-6 bg-zinc-950 flex justify-end gap-3">
				<button
					onclick={() => showCreateModal = false}
					class="px-4 py-2 rounded-lg hover:bg-zinc-900 transition-colors"
				>
					Cancel
				</button>
				<button
					onclick={createApp}
					disabled={creating || !newAppName}
					class="bg-blue-600 hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed text-white px-6 py-2 rounded-lg font-medium transition-colors"
				>
					{creating ? 'Creating...' : 'Create Application'}
				</button>
			</div>
		</div>
	</div>
{/if}

{#if showEditModal}
	<div class="fixed inset-0 bg-black/80 backdrop-blur-sm flex items-center justify-center p-4 z-50">
		<div class="bg-zinc-900 border border-zinc-800 rounded-xl w-full max-w-md overflow-hidden shadow-2xl">
			<div class="p-6 border-b border-zinc-800">
				<h2 class="text-xl font-bold">Edit Application</h2>
			</div>
			<div class="p-6 space-y-4">
				<div class="space-y-1">
					<label for="edit-name" class="text-sm font-medium text-zinc-400">App Name</label>
					<input
						id="edit-name"
						type="text"
						bind:value={editAppName}
						class="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-4 py-2 focus:outline-none focus:border-blue-500"
					/>
				</div>
				<div class="space-y-1">
					<label for="edit-desc" class="text-sm font-medium text-zinc-400">Description</label>
					<textarea
						id="edit-desc"
						bind:value={editAppDesc}
						class="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-4 py-2 focus:outline-none focus:border-blue-500 h-24 resize-none"
					></textarea>
				</div>
				<div class="space-y-1">
					<label for="edit-icon" class="text-sm font-medium text-zinc-400">Icon URL</label>
					<div class="flex gap-2">
						<div class="w-10 h-10 bg-zinc-950 border border-zinc-800 rounded-lg flex items-center justify-center shrink-0">
							{#if editAppIcon}
								<img src={editAppIcon} alt="Preview" class="w-full h-full object-cover rounded-lg" />
							{:else}
								<ImageIcon class="w-5 h-5 text-zinc-600" />
							{/if}
						</div>
						<input
							id="edit-icon"
							type="text"
							bind:value={editAppIcon}
							class="flex-1 bg-zinc-950 border border-zinc-800 rounded-lg px-4 py-2 focus:outline-none focus:border-blue-500"
						/>
					</div>
				</div>
			</div>
			<div class="p-6 bg-zinc-950 flex justify-end gap-3">
				<button
					onclick={() => showEditModal = false}
					class="px-4 py-2 rounded-lg hover:bg-zinc-900 transition-colors"
				>
					Cancel
				</button>
				<button
					onclick={updateApp}
					disabled={updating || !editAppName}
					class="bg-blue-600 hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed text-white px-6 py-2 rounded-lg font-medium transition-colors"
				>
					{updating ? 'Updating...' : 'Update Application'}
				</button>
			</div>
		</div>
	</div>
{/if}

{#if showUploadModal && uploadingApp}
	<div class="fixed inset-0 bg-black/80 backdrop-blur-sm flex items-center justify-center p-4 z-50">
		<div class="bg-zinc-900 border border-zinc-800 rounded-xl w-full max-w-2xl overflow-hidden shadow-2xl max-h-[90vh] flex flex-col">
			<div class="p-6 border-b border-zinc-800">
				<h2 class="text-xl font-bold">New Release: {uploadingApp.name}</h2>
			</div>
			<div class="p-6 space-y-6 overflow-y-auto flex-1">
				<div class="space-y-1">
					<label for="version" class="text-sm font-medium text-zinc-400">Release Version</label>
					<input
						id="version"
						type="text"
						bind:value={releaseVersion}
						class="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-4 py-2 focus:outline-none focus:border-blue-500"
						placeholder="e.g. 1.0.1"
					/>
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
						onclick={() => { showUploadModal = false; uploadingApp = null; }}
						class="px-4 py-2 rounded-lg hover:bg-zinc-900 transition-colors"
					>
						Cancel
					</button>
					<button
						onclick={handleUpload}
						disabled={uploading || !releaseVersion || binaries.every(b => !b.file)}
						class="bg-blue-600 hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed text-white px-6 py-2 rounded-lg font-medium transition-colors"
					>
						{uploading ? 'Uploading...' : 'Create Release'}
					</button>
				</div>
			</div>
		</div>
	</div>
{/if}