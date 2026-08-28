<script lang="ts">
	import { onMount } from 'svelte';
	import { Alert, Badge, EmptyState, PageHeader, Skeleton, Table, icons } from '@facile/muse';
	import { backend, type Incident, type Monitor } from '$lib/backend';

	let incidents = $state<Incident[]>([]);
	let monitors = $state<Monitor[]>([]);
	let loading = $state(true);
	let error = $state('');

	onMount(async () => {
		try {
			[incidents, monitors] = await Promise.all([backend.listIncidents(), backend.listMonitors()]);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load incidents.';
		} finally {
			loading = false;
		}
	});

	const monitorOf = (id: number) => monitors.find((monitor) => monitor.id === id);

	function duration(incident: Incident) {
		const end = incident.resolved_at ? new Date(incident.resolved_at) : new Date();
		const minutes = Math.max(
			0,
			Math.round((end.getTime() - new Date(incident.opened_at).getTime()) / 60000)
		);
		if (minutes < 60) return `${minutes} min`;
		const hours = Math.floor(minutes / 60);
		if (hours < 24) return `${hours} h ${minutes % 60} min`;
		return `${Math.floor(hours / 24)} d ${hours % 24} h`;
	}
</script>

<svelte:head>
	<title>Incidents — Sonde</title>
</svelte:head>

<div class="flex flex-col gap-10">
	<PageHeader title="Incidents" description="Everything that went wrong, newest first." />

	{#if error}
		<Alert tone="danger" title="Could not load incidents">{error}</Alert>
	{:else if loading}
		<div class="flex flex-col gap-2">
			<Skeleton class="h-10 w-full" />
			<Skeleton class="h-10 w-full" />
			<Skeleton class="h-10 w-full" />
		</div>
	{:else if incidents.length === 0}
		<EmptyState
			icon={icons.shield}
			title="No incidents"
			description="When a monitor fails three checks in a row, it shows up here."
		/>
	{:else}
		<Table>
			<thead>
				<tr>
					<th scope="col">State</th>
					<th scope="col">Monitor</th>
					<th scope="col">Cause</th>
					<th scope="col">Opened</th>
					<th scope="col">Lasted</th>
				</tr>
			</thead>
			<tbody>
				{#each incidents as incident (incident.id)}
					{@const monitor = monitorOf(incident.monitor_id)}
					<tr>
						<td>
							<Badge tone={incident.resolved_at ? 'neutral' : 'danger'}>
								{incident.resolved_at ? 'resolved' : 'ongoing'}
							</Badge>
						</td>
						<td>
							{#if monitor}
								<a href={`/monitors/${monitor.id}`} class="font-medium text-fc-fg hover:underline">
									{monitor.name}
								</a>
							{:else}
								<span class="text-fc-fg-muted">—</span>
							{/if}
						</td>
						<td class="max-w-sm truncate">{incident.cause || '—'}</td>
						<td class="text-fc-fg-muted">{new Date(incident.opened_at).toLocaleString()}</td>
						<td class="text-fc-fg-muted">{duration(incident)}</td>
					</tr>
				{/each}
			</tbody>
		</Table>
	{/if}
</div>
