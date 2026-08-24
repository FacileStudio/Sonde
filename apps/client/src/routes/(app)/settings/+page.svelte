<script lang="ts">
	import { onMount } from 'svelte';
	import { Card, PageHeader, Section } from '@facile/muse';
	import { backend } from '$lib/backend';

	let exportError = $state('');

	async function download() {
		exportError = '';
		try {
			await backend.exportConfig();
		} catch (err) {
			exportError = err instanceof Error ? err.message : 'Export failed.';
		}
	}

	async function logout() {
		await backend.logout().catch(() => undefined);
		window.location.href = '/login';
	}
</script>

<svelte:head>
	<title>Settings — Sonde</title>
</svelte:head>

<PageHeader title="Settings" description="Instance housekeeping." />

<div class="flex max-w-2xl flex-col gap-6">
	<Section title="Config export">
		<Card class="p-4">
			<p class="mb-3 text-fc-sm text-fc-fg-muted">
				Snapshot of every monitor and status page as JSON — no secrets included. Keep it in git.
			</p>
			{#if exportError}
				<div class="mb-3 text-fc-sm text-fc-danger">{exportError}</div>
			{/if}
			<button
				class="h-9 rounded-md border border-fc-border px-4 text-fc-sm font-medium text-fc-fg hover:bg-fc-surface-hover"
				onclick={download}
			>
				Download sonde-config.json
			</button>
		</Card>
	</Section>

	<Section title="Session">
		<Card class="p-4">
			<button
				class="h-9 rounded-md bg-fc-danger px-4 text-fc-sm font-medium text-white hover:opacity-90"
				onclick={logout}
			>
				Log out
			</button>
		</Card>
	</Section>
</div>
