<script lang="ts">
	import { onMount } from 'svelte';
	import { Badge, Card, EmptyState, PageHeader, Spinner } from '@facile/muse';
	import { backend, type Incident } from '$lib/backend';

	let incidents = $state<Incident[] | null>(null);
	let error = $state('');

	onMount(async () => {
		try {
			incidents = await backend.listIncidents();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load incidents.';
		}
	});
</script>

<svelte:head>
	<title>Incidents — Sonde</title>
</svelte:head>

<PageHeader title="Incidents" description="Everything that went wrong, newest first." />

{#if error}
	<div class="rounded-md border border-fc-border bg-fc-danger/10 p-3 text-fc-sm text-fc-danger">
		{error}
	</div>
{:else if !incidents}
	<div class="flex justify-center py-16"><Spinner /></div>
{:else if incidents.length === 0}
	<EmptyState
		icon="alert"
		title="No incidents"
		description="When a monitor fails three checks in a row, it shows up here."
	/>
{:else}
	<Card class="p-0">
		<ul class="divide-y divide-fc-border">
			{#each incidents as incident (incident.id)}
				<li class="flex items-center gap-3 px-4 py-3">
					<Badge tone={incident.resolved_at ? 'neutral' : 'danger'}>
						{incident.resolved_at ? 'resolved' : 'ongoing'}
					</Badge>
					<div class="min-w-0 flex-1">
						<div class="truncate text-fc-sm text-fc-fg">{incident.cause}</div>
						<div class="text-fc-xs text-fc-fg-muted">
							monitor #{incident.monitor_id} · opened {new Date(incident.opened_at).toLocaleString()}
							{#if incident.resolved_at}
								· resolved {new Date(incident.resolved_at).toLocaleString()}
							{/if}
						</div>
					</div>
				</li>
			{/each}
		</ul>
	</Card>
{/if}
