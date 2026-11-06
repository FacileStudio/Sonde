<script lang="ts">
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
	import { clock, dateTime, percent as formatPercent } from '$lib/format';
	import { t } from '$lib/i18n/index.svelte';

	let data = $state<PublicStatusPage | null>(null);
	let loading = $state(true);
	let error = $state('');
	let fetchedAt = $state<Date | null>(null);

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
			{ label: t('monitor.status.up'), value: up },
			{ label: t('monitor.status.down'), value: down },
			{ label: t('public.slice.unknown'), value: unknown }
		].filter((slice) => slice.value > 0)
	);

	const tone = (monitor: PublicMonitor) =>
		monitor.current_status === 'up' ? 'success' : monitor.current_status === 'down' ? 'danger' : 'neutral';

	const REFRESH_MS = 60_000;

	async function load(target: string) {
		try {
			data = await backend.publicStatusPage(target);
			error = '';
			fetchedAt = new Date();
		} catch (err) {
			error = err instanceof Error ? err.message : t('public.notFound.description');
		} finally {
			loading = false;
		}
	}

	/*
	 * Keyed on the slug, not run once on mount: SvelteKit reuses this component across
	 * /status/a -> /status/b, so a one-shot fetch left b showing a's monitors. The interval is
	 * what makes the "live" claim in the header true on a wall display nobody reloads.
	 */
	$effect(() => {
		const target = slug;
		loading = true;
		data = null;
		void load(target);
		const timer = setInterval(() => void load(target), REFRESH_MS);
		return () => clearInterval(timer);
	});
</script>

<svelte:head>
	<title>{data ? t('public.title', { title: data.title }) : t('public.titleFallback')}</title>
</svelte:head>

<div class="min-h-dvh bg-fc-page px-4 py-10">
	<div class="mx-auto flex w-full max-w-3xl flex-col gap-10">
		<!-- `&& !data` because a background refresh that fails must not replace a page that is
		     rendering fine with "Page not found". The header keeps saying when it last read. -->
		{#if error && !data}
			<EmptyState
				icon={icons.globe}
				title={t('public.notFound.title')}
				description={t('public.notFound.description')}
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
					{fetchedAt
						? t('public.readAt', { time: clock(fetchedAt) })
						: t('public.refreshed')}
				</p>
			</header>

			{#if monitors.length === 0}
				<!-- The verdict only exists when there is something to judge. With no monitors,
				     down and unknown are both 0 and the success branch cheerfully announced
				     "0 of 0 services answering normally" above the empty state. -->
				<EmptyState
					icon={icons.monitor}
					title={t('public.empty.title')}
					description={t('public.empty.description')}
				/>
			{:else}
				<Alert
					tone={down > 0 ? 'danger' : unknown > 0 ? 'warning' : 'success'}
					title={down > 0
						? t('public.verdict.down')
						: unknown > 0
							? t('public.verdict.unknown')
							: t('public.verdict.ok')}
				>
					{down > 0
						? t('public.verdict.downText', { down, total: monitors.length })
						: t('public.verdict.okText', { up, total: monitors.length })}
				</Alert>

				<div class="grid grid-cols-1 gap-4 lg:grid-cols-3">
					<StatCard label={t('public.stat.operational')} value={`${up} / ${monitors.length}`} />
					<StatCard
						label={t('public.stat.openIncidents')}
						value={openIncidents}
						delta={openIncidents === 0 ? t('incidents.stat.nothingOngoing') : t('public.stat.beingWorked')}
					/>
					<StatCard
						label={t('public.stat.average')}
						value={formatPercent(average24h)}
						delta={measured.length === 0
							? t('monitor.noChecks')
							: t('public.stat.across', { count: measured.length })}
					/>
				</div>

				<div class="grid grid-cols-1 gap-4 lg:grid-cols-3">
					<Card class="flex flex-col gap-4 lg:col-span-1">
						<span class="text-fc-xs tracking-wide text-fc-fg-muted uppercase">{t('public.byState')}</span>
						<DonutChart
							data={slices}
							class="flex-1"
							size={180}
							centerValue={`${up}/${monitors.length}`}
							centerLabel={t('public.operational')}
						/>
					</Card>

					<div class="flex flex-col gap-4 lg:col-span-2">
						{#each monitors as monitor (monitor.id)}
							<Card class="flex flex-col gap-3">
								<div class="flex min-w-0 flex-wrap items-center gap-3">
									<StatusDot tone={tone(monitor)} pulse={monitor.current_status === 'down'} />
									<span class="min-w-0 flex-1 truncate font-medium text-fc-fg">{monitor.name}</span>
									<Badge tone="neutral">{t('public.badge24', { percent: formatPercent(monitor.uptime_24h) })}</Badge>
									<Badge tone="neutral">{t('public.badge7', { percent: formatPercent(monitor.uptime_7d) })}</Badge>
								</div>
								{#if monitor.open_incidents.length > 0}
									<ul class="flex flex-col gap-1 pt-3">
										{#each monitor.open_incidents as incident (incident.id)}
											<li class="text-fc-xs text-fc-danger">
												{t('public.failingSince', {
													cause: incident.cause || t('public.failingChecks'),
													when: dateTime(incident.opened_at)
												})}
											</li>
										{/each}
									</ul>
								{/if}
							</Card>
						{/each}
					</div>
				</div>
			{/if}

			<p class="text-center text-fc-xs text-fc-fg-muted">{t('public.poweredBy')}</p>
		{/if}
	</div>
</div>
