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
	import { dateTime, duration, shortDay } from '$lib/format';
	import { t } from '$lib/i18n/index.svelte';

	const ranges = $derived([
		{ id: '7', label: t('incidents.range.7') },
		{ id: '30', label: t('incidents.range.30') },
		{ id: '90', label: t('incidents.range.90') }
	]);

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
			error = err instanceof Error ? err.message : t('incidents.loadFailed');
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
			error = err instanceof Error ? err.message : t('incidents.statsFailed');
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
		{ name: t('incidents.series.opened'), data: (stats?.by_day ?? []).map((entry) => entry.opened), color: chartColor(0) }
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
		if (rest > 0) top.push({ label: t('incidents.other'), value: rest, color: chartColor(5) });
		return top;
	});
</script>

<svelte:head>
	<title>{t('nav.incidents')} · Sonde</title>
</svelte:head>

<div class="flex flex-col gap-10">
	<PageHeader title={t('nav.incidents')} description={t('incidents.description')} />

	{#if error}
		<Alert tone="danger" title={t('incidents.loadError')}>{error}</Alert>
	{:else if loading}
		<div class="flex flex-col gap-4">
			<Skeleton class="h-10 w-64" />
			<Skeleton class="h-24 w-full" />
			<Skeleton class="h-64 w-full" />
		</div>
	{:else}
		<Tabs items={ranges} value={range} onChange={changeRange} label={t('incidents.window')} />

		{#if statsLoading || !stats}
			<Skeleton class="h-24 w-full" />
		{:else}
			<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
				<StatCard
					label={t('incidents.stat.openNow')}
					value={stats.open}
					delta={stats.open === 0 ? t('incidents.stat.nothingOngoing') : t('incidents.stat.unresolved')}
				/>
				<StatCard
					label={t('incidents.stat.total', { days: stats.days })}
					value={stats.total}
					delta={t('incidents.stat.resolved', { count: stats.resolved })}
				/>
				<StatCard
					label="MTTR"
					value={duration(stats.mttr_seconds)}
					delta={t('incidents.stat.mttrHint')}
				/>
				<StatCard
					label={t('incidents.stat.downtime')}
					value={duration(stats.total_downtime_seconds)}
					delta={stats.longest_seconds === null
						? t('incidents.stat.noOutage')
						: t('incidents.stat.longest', { duration: duration(stats.longest_seconds) })}
				/>
			</div>
		{/if}

		{#if stats}
			<div class="grid gap-4 lg:grid-cols-3">
				<Card class="flex flex-col gap-4 lg:col-span-2">
					<div class="flex flex-col gap-1">
						<h2 class="text-fc-md font-semibold text-fc-fg">{t('incidents.overTime')}</h2>
						<p class="text-fc-xs text-fc-fg-muted">{t('incidents.perDay')}</p>
					</div>
					<BarChart
						class="flex-1"
						series={openedSeries}
						labels={dayLabels}
						height={220}
						emptyLabel={t('incidents.noneInWindow')}
					/>
				</Card>

				{#if load.length > 0}
					<Card class="flex flex-col gap-4">
						<div class="flex flex-col gap-1">
							<h2 class="text-fc-md font-semibold text-fc-fg">{t('incidents.pain')}</h2>
							<p class="text-fc-xs text-fc-fg-muted">{t('incidents.byMonitor')}</p>
						</div>
						<DonutChart
							class="flex-1"
							data={load}
							centerLabel={t('incidents.downtimeLabel')}
							centerValue={duration(stats.total_downtime_seconds)}
							valueFormat={(value: number) => duration(value)}
						/>
					</Card>
				{/if}
			</div>
		{/if}

		<Section title={t('incidents.all')}>
			{#if incidents.length === 0}
				<EmptyState
					icon={icons.shield}
					title={t('incidents.empty.title')}
					description={t('incidents.empty.description')}
				/>
			{:else}
				<Table>
					<thead>
						<tr>
							<th scope="col">{t('incidents.col.state')}</th>
							<th scope="col">{t('monitors.col.monitor')}</th>
							<th scope="col">{t('incidents.col.cause')}</th>
							<th scope="col">{t('incidents.col.opened')}</th>
							<th scope="col">{t('incidents.col.resolved')}</th>
							<th scope="col">{t('incidents.col.lasted')}</th>
						</tr>
					</thead>
					<tbody>
						{#each incidents as incident (incident.id)}
							{@const monitor = monitorOf(incident.monitor_id)}
							<tr>
								<td>
									<Badge tone={incident.resolved_at ? 'neutral' : 'danger'}>
										{incident.resolved_at ? t('incidents.state.resolved') : t('incidents.state.ongoing')}
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
									{dateTime(incident.opened_at)}
								</td>
								<td class="font-fc-mono text-fc-fg-muted">
									{incident.resolved_at ? dateTime(incident.resolved_at) : '—'}
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
