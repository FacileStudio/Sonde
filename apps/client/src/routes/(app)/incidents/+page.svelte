<script lang="ts">
	import { onMount } from 'svelte';
	import {
		Alert,
		Badge,
		BarChart,
		Card,
		DonutChart,
		EmptyState,
		PageHeader,
		Section,
		Skeleton,
		StatCard,
		Table,
		Tabs,
		chartColor,
		icons
	} from '@facile/muse';
	import { backend, type Incident, type IncidentStats, type Monitor } from '$lib/backend';
	import { duration, shortDay } from '$lib/format';

	const ranges = [
		{ id: '7', label: '7 days' },
		{ id: '30', label: '30 days' },
		{ id: '90', label: '90 days' }
	];

	let incidents = $state<Incident[]>([]);
	let monitors = $state<Monitor[]>([]);
	let stats = $state<IncidentStats | null>(null);
	let range = $state('30');
	let loading = $state(true);
	let statsLoading = $state(false);
	let error = $state('');

	onMount(async () => {
		try {
			[incidents, monitors, stats] = await Promise.all([
				backend.listIncidents(),
				backend.listMonitors(),
				backend.incidentStats(Number(range))
			]);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load incidents.';
		} finally {
			loading = false;
		}
	});

	async function changeRange(id: string) {
		range = id;
		statsLoading = true;
		try {
			stats = await backend.incidentStats(Number(id));
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load incident stats.';
		} finally {
			statsLoading = false;
		}
	}

	const monitorOf = (id: number) => monitors.find((monitor) => monitor.id === id);

	function lasted(incident: Incident): number {
		const end = incident.resolved_at ? new Date(incident.resolved_at) : new Date();
		return (end.getTime() - new Date(incident.opened_at).getTime()) / 1000;
	}

	const dayLabels = $derived((stats?.by_day ?? []).map((entry) => shortDay(entry.day)));
	const openedSeries = $derived([
		{ name: 'Opened', data: (stats?.by_day ?? []).map((entry) => entry.opened), color: chartColor(0) }
	]);

	/* The palette wraps after six, so a seventh monitor would repaint an earlier
	   one's colour and read as the same service twice. */
	const load = $derived.by(() => {
		const rows = stats?.by_monitor ?? [];
		const top = rows.slice(0, 5).map((row, index) => ({
			label: row.name,
			value: row.downtime_seconds,
			color: chartColor(index)
		}));
		const rest = rows.slice(5).reduce((sum, row) => sum + row.downtime_seconds, 0);
		if (rest > 0) top.push({ label: 'Other', value: rest, color: chartColor(5) });
		return top;
	});
</script>

<svelte:head>
	<title>Incidents — Sonde</title>
</svelte:head>

<div class="flex flex-col gap-10">
	<PageHeader title="Incidents" description="Everything that went wrong, newest first." />

	{#if error}
		<Alert tone="danger" title="Could not load incidents">{error}</Alert>
	{:else if loading}
		<div class="flex flex-col gap-4">
			<Skeleton class="h-10 w-64" />
			<Skeleton class="h-24 w-full" />
			<Skeleton class="h-64 w-full" />
		</div>
	{:else}
		<Tabs items={ranges} value={range} onChange={changeRange} label="Incident window" />

		{#if statsLoading || !stats}
			<Skeleton class="h-24 w-full" />
		{:else}
			<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
				<StatCard
					label="Open now"
					value={stats.open}
					delta={stats.open === 0 ? 'Nothing ongoing' : 'Still unresolved'}
				/>
				<StatCard
					label={`Incidents, ${stats.days} days`}
					value={stats.total}
					delta={`${stats.resolved} resolved`}
				/>
				<StatCard
					label="MTTR"
					value={duration(stats.mttr_seconds)}
					delta="Resolved incidents only — an open one has no repair time yet"
				/>
				<StatCard
					label="Total downtime"
					value={duration(stats.total_downtime_seconds)}
					delta={stats.longest_seconds === null
						? 'No outage recorded'
						: `Longest ${duration(stats.longest_seconds)}`}
				/>
			</div>
		{/if}

		{#if stats}
			<div class="grid gap-4 lg:grid-cols-3">
				<Card class="flex flex-col gap-4 lg:col-span-2">
					<div class="flex flex-col gap-1">
						<h2 class="text-fc-md font-semibold text-fc-fg">Incidents over time</h2>
						<p class="text-fc-xs text-fc-fg-muted">Incidents opened per day</p>
					</div>
					<BarChart
						class="flex-1"
						series={openedSeries}
						labels={dayLabels}
						height={220}
						emptyLabel="No incidents in this window"
					/>
				</Card>

				{#if load.length > 0}
					<Card class="flex flex-col gap-4">
						<div class="flex flex-col gap-1">
							<h2 class="text-fc-md font-semibold text-fc-fg">Where the pain is</h2>
							<p class="text-fc-xs text-fc-fg-muted">Downtime by monitor</p>
						</div>
						<DonutChart
							class="flex-1"
							data={load}
							centerLabel="Downtime"
							centerValue={duration(stats.total_downtime_seconds)}
							valueFormat={(value: number) => duration(value)}
						/>
					</Card>
				{/if}
			</div>
		{/if}

		<Section title="All incidents">
			{#if incidents.length === 0}
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
							<th scope="col">Resolved</th>
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
										<a
											href={`/monitors/${monitor.id}`}
											class="font-medium text-fc-fg underline underline-offset-4 hover:text-fc-accent"
										>
											{monitor.name}
										</a>
									{:else}
										<span class="text-fc-fg-muted">—</span>
									{/if}
								</td>
								<td class="max-w-sm truncate">{incident.cause || '—'}</td>
								<td class="font-fc-mono text-fc-fg-muted">
									{new Date(incident.opened_at).toLocaleString()}
								</td>
								<td class="font-fc-mono text-fc-fg-muted">
									{incident.resolved_at ? new Date(incident.resolved_at).toLocaleString() : '—'}
								</td>
								<td class="font-fc-mono text-fc-fg-muted">{duration(lasted(incident))}</td>
							</tr>
						{/each}
					</tbody>
				</Table>
			{/if}
		</Section>
	{/if}
</div>
