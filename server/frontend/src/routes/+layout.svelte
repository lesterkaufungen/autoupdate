<script lang="ts">
	import { onMount } from 'svelte';
	import '../app.css';
	import { LayoutDashboard, Box, Upload, BarChart3, List, LogOut, Sun, Moon } from 'lucide-svelte';
	import { api } from '$lib/api';

	let { children } = $props();

	let adminToken = $state('');
	let username = $state('');
	let password = $state('');
	let loggingIn = $state(false);
	let loginError = $state('');
	let theme = $state('dark');

	onMount(() => {
		adminToken = localStorage.getItem('admin_token') || '';
		theme = localStorage.getItem('theme') || (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light');
		applyTheme();
	});

	function applyTheme() {
		if (theme === 'dark') {
			document.documentElement.classList.add('dark');
		} else {
			document.documentElement.classList.remove('dark');
		}
		localStorage.setItem('theme', theme);
	}

	function toggleTheme() {
		theme = theme === 'dark' ? 'light' : 'dark';
		applyTheme();
	}

	async function handleLogin() {
		if (!username || !password) return;
		loggingIn = true;
		loginError = '';
		try {
			const res = await api.login(username, password);
			adminToken = res.token;
			localStorage.setItem('admin_token', adminToken);
		} catch (e: any) {
			loginError = e.message;
		} finally {
			loggingIn = false;
		}
	}

	function logout() {
		localStorage.removeItem('admin_token');
		adminToken = '';
		window.location.reload();
	}

	const navItems = [
		{ href: '/ui', label: 'Dashboard', icon: LayoutDashboard },
		{ href: '/ui/apps', label: 'Applications', icon: Box },
		{ href: '/ui/releases', label: 'Releases', icon: Upload },
		{ href: '/ui/stats', label: 'Statistics', icon: BarChart3 },
		{ href: '/ui/logs', label: 'Logs', icon: List },
	];
</script>

<svelte:head>
	<link rel="icon" href="/ui/favicon.svg" />
	<title>autoupdate</title>
</svelte:head>

{#if !adminToken}
	<div class="flex h-screen items-center justify-center bg-zinc-950 text-zinc-100 p-4 transition-colors duration-200">
		<div class="w-full max-w-md bg-zinc-900 border border-zinc-800 rounded-2xl shadow-2xl overflow-hidden">
			<div class="p-8 text-center border-b border-zinc-800 bg-zinc-950/50">
				<div class="w-16 h-16 bg-blue-600 rounded-2xl flex items-center justify-center mx-auto mb-4 shadow-lg shadow-blue-900/20 overflow-hidden">
					<img src="/ui/favicon.svg" alt="logo" class="w-10 h-10" />
				</div>
				<h1 class="text-2xl font-bold tracking-tight">Welcome back</h1>
				<p class="text-zinc-500 text-sm mt-1">Sign in to manage your updates</p>
			</div>
			
			<form onsubmit={(e) => { e.preventDefault(); handleLogin(); }} class="p-8 space-y-5">
				{#if loginError}
					<div class="p-3 bg-red-900/20 border border-red-900/50 rounded-lg text-red-500 text-sm text-center animate-shake">
						{loginError}
					</div>
				{/if}

				<div class="space-y-1.5">
					<label for="username" class="text-xs font-bold text-zinc-500 uppercase tracking-widest ml-1">Username</label>
					<input
						id="username"
						type="text"
						bind:value={username}
						class="w-full bg-zinc-950 border border-zinc-800 rounded-xl px-4 py-3 focus:outline-none focus:border-blue-500 transition-all"
						placeholder="admin"
						required
					/>
				</div>
				<div class="space-y-1.5">
					<label for="password" class="text-xs font-bold text-zinc-500 uppercase tracking-widest ml-1">Password</label>
					<input
						id="password"
						type="password"
						bind:value={password}
						class="w-full bg-zinc-950 border border-zinc-800 rounded-xl px-4 py-3 focus:outline-none focus:border-blue-500 transition-all"
						placeholder="••••••••"
						required
					/>
				</div>
				<button
					type="submit"
					disabled={loggingIn}
					class="w-full bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white font-bold py-3 rounded-xl transition-all shadow-lg shadow-blue-900/20 flex items-center justify-center gap-2"
				>
					{#if loggingIn}
						<div class="w-5 h-5 border-2 border-white/30 border-t-white rounded-full animate-spin"></div>
					{:else}
						Sign In
					{/if}
				</button>
			</form>

			<div class="p-4 bg-zinc-950/50 border-t border-zinc-800 flex justify-center">
				<button 
					onclick={toggleTheme}
					class="p-2 rounded-lg hover:bg-zinc-800 transition-colors text-zinc-500"
					aria-label="Toggle theme"
				>
					{#if theme === 'dark'}
						<Sun class="w-5 h-5" />
					{:else}
						<Moon class="w-5 h-5" />
					{/if}
				</button>
			</div>
		</div>
	</div>
{:else}
	<div class="flex h-screen bg-zinc-950 text-zinc-100 transition-colors duration-200">
		<!-- Sidebar -->
		<aside class="w-64 border-r border-zinc-800 flex flex-col">
			<div class="p-6 border-b border-zinc-800 flex items-center gap-3">
				<div class="w-8 h-8 bg-blue-600 rounded-lg flex items-center justify-center overflow-hidden">
					<img src="/ui/favicon.svg" alt="logo" class="w-5 h-5" />
				</div>
				<span class="font-bold text-xl tracking-tight">autoupdate</span>
			</div>

			<nav class="flex-1 p-4 space-y-2">
				{#each navItems as item}
					<a
						href={item.href}
						class="flex items-center gap-3 px-3 py-2 rounded-md transition-colors hover:bg-zinc-800"
					>
						<item.icon class="w-5 h-5" />
						<span>{item.label}</span>
					</a>
				{/each}
			</nav>

			<div class="p-4 border-t border-zinc-800 space-y-2">
				<button 
					onclick={toggleTheme}
					class="w-full flex items-center gap-3 px-3 py-2 rounded-md text-zinc-500 hover:text-zinc-100 hover:bg-zinc-800 transition-all"
				>
					{#if theme === 'dark'}
						<Sun class="w-5 h-5" />
						<span>Light Mode</span>
					{:else}
						<Moon class="w-5 h-5" />
						<span>Dark Mode</span>
					{/if}
				</button>

				<button
					onclick={logout}
					class="w-full flex items-center gap-3 px-3 py-2 rounded-md text-zinc-500 hover:text-red-400 hover:bg-zinc-800 transition-all"
				>
					<LogOut class="w-5 h-5" />
					<span>Logout</span>
				</button>
			</div>
		</aside>

		<!-- Main Content -->
		<main class="flex-1 overflow-auto p-8">
			<div class="max-w-6xl mx-auto">
				{@render children()}
			</div>
		</main>
	</div>
{/if}


