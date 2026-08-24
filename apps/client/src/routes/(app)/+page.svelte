<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { Badge, Button, Card, EmptyState, PageHeader, Spinner, StatusDot } from '@facile/muse';
	import { backend, type Monitor } from '$lib/backend';

	let monitors = $state<Monitor[] | null>(null);
	let error = $state('');

	const dotTone = (m: Monitor) => (m.enabled ? 'success' : 'neutral');

	onMount(async () => {
		try {
			monitors = await backend.listMonitors();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load monitors.';
		}
	});
</script>

<svelte:head>
	<title>Monitors — Sonde</title>
</svelte:head>

<PageHeader
	title="Monitors"
	description="What Sonde watches, and whether it is alive right now."
>
	{#snippet actions()}
		<Button onclick={() => goto('/monitors/new')}>New monitor</Button>
	{/snippet}
</PageHeader>

{#if error}
	<div class="rounded-md border border-fc-border bg-fc-danger/10 p-3 text-fc-sm text-fc-danger">
		{error}
	</div>
{:else if !monitors}
	<div class="flex justify-center py-16"><Spinner /></div>
{:else if monitors.length === 0}
	<EmptyState
		icon="activity"
		title="No monitors yet"
		description="Add an HTTP endpoint, a TCP port, or a push heartbeat to start watching."
	>
		<Button variant="outline" onclick={() => goto('/monitors/new')}>Create your first monitor</Button>
	</EmptyState>
{:else}
	<div class="flex flex-col gap-2">
		{#each monitors as monitor (monitor.id)}
			<Card class="cursor-pointer p-4 transition-colors hover:bg-fc-surface-hover" onclick={() => goto(`/monitors/${monitor.id}`)}>
				<div class="flex items-center gap-3">
					<StatusDot tone={dotTone(monitor)} pulse={monitor.enabled} />
					<div class="min-w-0 flex-1">
						<div class="flex items-center gap-2">
							<span class="truncate font-medium text-fc-fg">{monitor.name}</span>
							<Badge tone="neutral">{monitor.type}</Badge>
							{#if !monitor.enabled}
								<Badge tone="warning">paused</Badge>
							{/if}
						</div>
						<div class="truncate text-fc-xs text-fc-fg-muted">
							{monitor.target || `push token ${monitor.push_token ? '••••' + monitor.push_token.slice(-4) : '—'} · every ${monitor.interval_seconds}s`}
						</div>
					</div>
				</div>
			</Card>
		{/each}
	</div>
{/if}
