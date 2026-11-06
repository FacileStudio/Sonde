<script lang="ts">
	/*
	 * One monitor's history: coverage, latency and failures over a chosen
	 * window, read from the rollup tables rather than from raw checks.
	 *
	 * The window lives in local state and not in the URL. That is the opposite
	 * of the settings tabs (CHARTE §14), and deliberately: a settings section is
	 * a place you link someone to, a chart range is a thing you scrub while
	 * looking at one page.
	 */
	import { onMount } from 'svelte';
	import { BarChart, Card, LineChart, Section, Skeleton, Tabs, chartColor } from '@facile/muse';
	import UptimeBar from '$lib/components/UptimeBar.svelte';
	import { backend, UPTIME_WINDOWS, type MonitorHistory, type UptimeWindow } from '$lib/backend';
	import { latencySeries, milliseconds, shortDay, shortTime } from '$lib/format';
	import { t } from '$lib/i18n/index.svelte';

	let { monitorId, onError }: { monitorId: number; onError?: (message: string) => void } = $props();

	let history = $state<MonitorHistory | null>(null);
	let range = $state<UptimeWindow>('24h');
	let loading = $state(true);

	const rangeTabs = UPTIME_WINDOWS.map((entry) => ({ id: entry, label: entry }));

	const buckets = $derived(history?.buckets ?? []);
	const resolution = $derived(history?.resolution ?? 'hour');
	const labels = $derived(
		buckets.map((bucket) =>
			resolution === 'day' ? shortDay(bucket.bucket_start) : shortTime(bucket.bucket_start)
		)
	);
	const latency = $derived([
		{ name: t('history.avgLatency'), data: latencySeries(buckets), color: chartColor(0) }
	]);
	/* One "failed checks" series, not failed stacked on successful: a healthy
	   monitor reports ~60 successes to a single failure per bucket, and the
	   failed segment of that stack is under a pixel — the one thing the chart
	   exists to show. */
	const failures = $derived([
		{ name: t('history.failedChecks'), data: buckets.map((bucket) => bucket.failed), color: chartColor(3) }
	]);

	async function load(window: UptimeWindow) {
		loading = true;
		try {
			history = await backend.monitorHistory(monitorId, window);
		} catch (err) {
			onError?.(err instanceof Error ? err.message : t('history.loadFailed'));
		} finally {
			loading = false;
		}
	}

	function selectRange(id: string) {
		range = id as UptimeWindow;
		void load(range);
	}

	onMount(() => load(range));
</script>

<Section title={t('history.title')} description={t('history.description')}>
	{#snippet actions()}
		<Tabs items={rangeTabs} value={range} onChange={selectRange} label={t('history.window')} />
	{/snippet}

	{#if loading}
		<Skeleton class="h-64 w-full" />
	{:else}
		<div class="flex flex-col gap-4">
			<Card class="p-5">
				<UptimeBar {buckets} {resolution} height={32} label={t('history.uptimeHistory')} />
			</Card>

			<div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
				<Card class="flex flex-col gap-4 p-5">
					<div>
						<h3 class="text-fc-md font-semibold text-fc-fg">{t('history.response')}</h3>
						<p class="text-fc-xs text-fc-fg-muted">{resolution === 'day' ? t('history.latencyDay') : t('history.latencyHour')}</p>
					</div>
					<LineChart
						class="flex-1"
						series={latency}
						{labels}
						area
						height={220}
						yFormat={(n) => milliseconds(n)}
						emptyLabel={t('history.noChecks')}
					/>
				</Card>

				<Card class="flex flex-col gap-4 p-5">
					<div>
						<h3 class="text-fc-md font-semibold text-fc-fg">{t('history.failedChecks')}</h3>
						<p class="text-fc-xs text-fc-fg-muted">{resolution === 'day' ? t('history.failedDay') : t('history.failedHour')}</p>
					</div>
					<BarChart
						class="flex-1"
						series={failures}
						{labels}
						height={220}
						emptyLabel={t('history.noChecks')}
					/>
				</Card>
			</div>
		</div>
	{/if}
</Section>
