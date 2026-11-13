<script lang="ts">
	import { Tooltip } from '@facile/muse';
	/*
	 * The uptime strip: one tick per bucket, green up, red down, hollow for a
	 * bucket nobody probed. muse ships no such primitive (CHARTE §12 covers
	 * line, bar, donut and sparkline), and a BarChart cannot express the third
	 * state — a zero-height bar and a bar that was never drawn look identical.
	 *
	 * It replaces MonitorStatusChart, which read raw checks and so could only
	 * ever show the last 200 probes: 3 h 20 min for a 60 s monitor, whatever
	 * window the page claimed. This reads the rollups, so 90 days is 90 ticks.
	 */
	import type { Bucket } from '$lib/backend';
	import { dateTime, duration, milliseconds, percent, shortDay, shortTime } from '$lib/format';
	import { t } from '$lib/i18n/index.svelte';

	let {
		buckets = [],
		height = 28,
		resolution = 'hour',
		label
	}: {
		buckets?: Bucket[];
		height?: number;
		resolution?: 'hour' | 'day';
		label?: string;
	} = $props();

	/* Failed, not down: a bucket is an hour or a day, so it can hold both
	   successes and failures. Anything short of clean is drawn as degraded, and
	   only a bucket that failed outright reads as an outage. */
	function tone(bucket: Bucket): 'ok' | 'degraded' | 'down' | 'empty' {
		if (bucket.total === 0) return 'empty';
		if (bucket.failed === 0) return 'ok';
		if (bucket.failed >= bucket.total) return 'down';
		return 'degraded';
	}

	const FILL = {
		ok: 'bg-fc-success',
		degraded: 'bg-fc-warning',
		down: 'bg-fc-danger',
		empty: 'bg-fc-field'
	} as const;

	function describe(bucket: Bucket): string {
		const when = resolution === 'day' ? shortDay(bucket.bucket_start) : shortTime(bucket.bucket_start);
		if (bucket.total === 0) return t('uptimeBar.notProbed', { when });
		return t('uptimeBar.describe', {
			when,
			uptime: percent(bucket.uptime, 1),
			total: bucket.total,
			latency: milliseconds(bucket.avg_latency_ms)
		});
	}

	const measured = $derived(buckets.filter((bucket) => bucket.total > 0));
	const covered = $derived(
		buckets.length === 0 ? 0 : Math.round((measured.length / buckets.length) * 100)
	);
	const outage = $derived(
		measured.reduce(
			(total, bucket) =>
				total + (bucket.failed / bucket.total) * (resolution === 'day' ? 86400 : 3600),
			0
		)
	);
</script>

<div class="flex min-w-0 flex-col gap-2">
	<!-- The strip is decoration; the table below it is the accessible
	     representation, the same split every muse chart makes (CHARTE §12).
	     The sr-only wrapper is a div, never the table itself: overflow:hidden
	     is ignored on a table box, so the class would leave the page with a
	     horizontal scrollbar. -->
	<div class="flex items-end gap-px" style="height: {height}px" aria-hidden="true">
		{#each buckets as bucket (bucket.bucket_start)}
			<Tooltip text={describe(bucket)} describe={false} class="contents">
				<div
					class="h-full min-w-[2px] flex-1 rounded-fc-xs transition-opacity hover:opacity-70 {FILL[
						tone(bucket)
					]} {tone(bucket) === 'empty' ? 'opacity-40' : ''}"
				></div>
			</Tooltip>
		{:else}
			<div class="h-full w-full rounded-fc-xs bg-fc-field opacity-40"></div>
		{/each}
	</div>

	{#if buckets.length > 0}
		<div class="flex items-center justify-between font-fc-mono text-fc-xs text-fc-fg-muted">
			<span>{t('uptimeBar.covered', { percent: percent(covered, 0) })}</span>
			<span>{outage > 0 ? t('uptimeBar.down', { duration: duration(outage) }) : t('uptimeBar.noDowntime')}</span>
		</div>
	{/if}

	<div class="sr-only">
		<table>
			<caption>{label ?? t('history.uptimeHistory')}</caption>
			<thead>
				<tr><th scope="col">{t('uptimeBar.bucket')}</th><th scope="col">{t('uptimeBar.uptime')}</th><th scope="col">{t('uptimeBar.checks')}</th></tr>
			</thead>
			<tbody>
				{#each buckets as bucket (bucket.bucket_start)}
					<tr>
						<td>{dateTime(bucket.bucket_start)}</td>
						<td>{bucket.total === 0 ? t('uptimeBar.notProbedShort') : percent(bucket.uptime, 1)}</td>
						<td>{bucket.total}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
</div>
