<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { SideBar, Button, Spinner } from '@facile/muse';
	import { backend, type AuthUser } from '$lib/backend';

	let { children } = $props();

	let user = $state<AuthUser | null>(null);
	let checked = $state(false);

	onMount(async () => {
		try {
			const res = await backend.me();
			user = res.user;
		} catch {
			goto(`/login?next=${encodeURIComponent(page.url.pathname)}`);
			return;
		}
		checked = true;
	});

	async function logout() {
		await backend.logout().catch(() => undefined);
		goto('/login');
	}

	const pages = [
		{ label: 'Monitors', href: '/', icon: 'activity' },
		{ label: 'Incidents', href: '/incidents', icon: 'alert' },
		{ label: 'Status pages', href: '/status-pages', icon: 'globe' },
		{ label: 'Settings', href: '/settings', icon: 'settings' }
	].map((entry) => ({ ...entry, active: page.url.pathname === entry.href }));
</script>

<svelte:head>
	<title>Sonde</title>
</svelte:head>

<div class="flex min-h-dvh bg-fc-page">
	<div class="hidden sm:block">
		<SideBar title="Sonde" icon="activity" {pages} user={user ? { name: user.email } : undefined} />
	</div>
	<div class="flex min-w-0 flex-1 flex-col">
		<header
			class="flex items-center justify-between gap-2 border-b border-fc-border px-4 py-3 sm:hidden"
		>
			<span class="font-semibold text-fc-fg">Sonde</span>
			<Button variant="ghost" size="sm" onclick={logout}>Log out</Button>
		</header>
		<nav class="flex overflow-x-auto border-b border-fc-border px-2 py-1 sm:hidden">
			{#each pages as entry (entry.href)}
				<a
					href={entry.href}
					class="shrink-0 px-3 py-2 text-fc-sm whitespace-nowrap {entry.active
						? 'text-fc-fg font-medium'
						: 'text-fc-fg-muted'}"
				>
					{entry.label}
				</a>
			{/each}
		</nav>
		<main class="min-w-0 flex-1 p-4 sm:p-6">
			{#if checked}
				{@render children()}
			{:else}
				<div class="flex h-full items-center justify-center">
					<Spinner />
				</div>
			{/if}
		</main>
	</div>
</div>
