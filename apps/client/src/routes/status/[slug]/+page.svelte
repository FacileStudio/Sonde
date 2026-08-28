<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import {
		Alert,
		Badge,
		Card,
		DonutChart,
		EmptyState,
		Skeleton,
		StatCard,
		StatusDot,
		icons
	} from '@facile/muse';
	import { backend, type PublicMonitor, type PublicStatusPage } from '$lib/backend';

	let data = $state<PublicStatusPage | null>(null);
	let loading = $state(true);
	let error = $state('');

	let slug = $derived(page.params.slug ?? '');

	const monitors = $derived(data?.monitors ?? []);
	const up = $derived(monitors.filter((monitor) => monitor.current_status === 'up').length);
	const down = $derived(monitors.filter((monitor) => monitor.current_status === 'down').length);
	const unknown = $derived(monitors.filter((monitor) => monitor.current_status === 'unknown').length);
	const openIncidents = $derived(
		monitors.reduce((total, monitor) => total + monitor.open_incidents.length, 0)
	);

	const measured = $derived(monitors.filter((monitor) => monitor.uptime_24h !== null));
	const average24h = $derived(
		measured.length === 0
			? null
			: measured.reduce((total, monitor) => total + (monitor.uptime_24h ?? 0), 0) / measured.length
	);

	const slices = $derived(
		[
			{ label: 'Up', value: up },
			{ label: 'Down', value: down },
			{ label: 'Unknown', value: unknown }
		].filter((slice) => slice.value > 0)
	);

	const tone = (monitor: PublicMonitor) =>
		monitor.current_status === 'up' ? 'success' : monitor.current_status === 'down' ? 'danger' : 'neutral';

	const percent = (value: number | null) => (value === null ? '—' : `${value.toFixed(2)}%`);

	onMount(async () => {
		try {
			data = await backend.publicStatusPage(slug);
		} catch (err) {
			error = err instanceof Error ? err.message : 'No public status page lives at this address.';
		} finally {
			loading = false;
		}
	});
</script>

<svelte:head>
	<title>{data ? `${data.title} — Status` : 'Status — Sonde'}</title>
</svelte:head>

<div class="min-h-dvh bg-fc-page px-4 py-10">
	<div class="mx-auto flex w-full max-w-3xl flex-col gap-10">
		{#if error}
			<EmptyState
				icon={icons.globe}
				title="Page not found"
				description="No public status page lives at this address."
			/>
		{:else if loading || !data}
			<div class="flex flex-col gap-4">
				<Skeleton class="h-8 w-48" />
				<Skeleton class="h-16 w-full" />
				<Skeleton class="h-24 w-full" />
			</div>
		{:else}
			<header class="flex flex-col gap-1 text-center">
				<h1 class="text-fc-2xl font-semibold text-fc-fg">{data.title}</h1>
				<p class="text-fc-sm text-fc-fg-muted">
					Live status, updated every time Sonde runs a check.
				</p>
			</header>

			<Alert
				tone={down > 0 ? 'danger' : unknown > 0 ? 'warning' : 'success'}
				title={down > 0
					? 'Some systems are down'
					: unknown > 0
						? 'Some systems have not reported yet'
						: 'All systems operational'}
			>
				{down > 0
					? `${down} of ${monitors.length} services are failing their checks. We are on it.`
					: `${up} of ${monitors.length} services are answering normally.`}
			</Alert>

			{#if monitors.length === 0}
				<EmptyState
					icon={icons.monitor}
					title="Nothing published yet"
					description="This status page has no monitors on it."
				/>
			{:else}
				<div class="grid grid-cols-1 gap-4 lg:grid-cols-3">
					<StatCard label="Operational" value={`${up} / ${monitors.length}`} />
					<StatCard
						label="Open incidents"
						value={openIncidents}
						delta={openIncidents === 0 ? 'Nothing ongoing' : 'Being worked on'}
					/>
					<StatCard
						label="Average uptime, 24h"
						value={average24h === null ? '—' : `${average24h.toFixed(2)}%`}
						delta={measured.length === 0
							? 'No checks recorded yet'
							: `Across ${measured.length} services`}
					/>
				</div>

				<div class="grid grid-cols-1 gap-4 lg:grid-cols-3">
					<Card class="flex flex-col gap-4 lg:col-span-1">
						<span class="text-fc-xs tracking-wide text-fc-fg-muted uppercase">Services by state</span>
						<DonutChart
							data={slices}
							class="flex-1"
							size={180}
							centerValue={`${up}/${monitors.length}`}
							centerLabel="operational"
						/>
					</Card>

					<div class="flex flex-col gap-4 lg:col-span-2">
						{#each monitors as monitor (monitor.id)}
							<Card class="flex flex-col gap-3">
								<div class="flex min-w-0 flex-wrap items-center gap-3">
									<StatusDot tone={tone(monitor)} pulse={monitor.current_status === 'down'} />
									<span class="min-w-0 flex-1 truncate font-medium text-fc-fg">{monitor.name}</span>
									<Badge tone="neutral">{percent(monitor.uptime_24h)} · 24h</Badge>
									<Badge tone="neutral">{percent(monitor.uptime_7d)} · 7d</Badge>
								</div>
								{#if monitor.open_incidents.length > 0}
									<ul class="flex flex-col gap-1 border-t border-fc-border pt-3">
										{#each monitor.open_incidents as incident (incident.id)}
											<li class="text-fc-xs text-fc-danger">
												{incident.cause || 'Failing checks'}, since
												{new Date(incident.opened_at).toLocaleString()}
											</li>
										{/each}
									</ul>
								{/if}
							</Card>
						{/each}
					</div>
				</div>
			{/if}

			<p class="text-center text-fc-xs text-fc-fg-muted">Powered by Sonde</p>
		{/if}
	</div>
</div>
