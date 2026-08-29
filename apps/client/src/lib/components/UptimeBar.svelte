<script lang="ts">
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
	import { duration, milliseconds, percent, shortDay, shortTime } from '$lib/format';

	let {
		buckets = [],
		height = 28,
		resolution = 'hour',
		label = 'Uptime history'
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
		empty: 'bg-fc-border'
	} as const;

	function describe(bucket: Bucket): string {
		const when = resolution === 'day' ? shortDay(bucket.bucket_start) : shortTime(bucket.bucket_start);
		if (bucket.total === 0) return `${when}: not probed`;
		return `${when}: ${percent(bucket.uptime, 1)} of ${bucket.total} checks, ${milliseconds(bucket.avg_latency_ms)} average`;
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
			<div
				class="h-full min-w-[2px] flex-1 rounded-fc-xs transition-opacity hover:opacity-70 {FILL[
					tone(bucket)
				]} {tone(bucket) === 'empty' ? 'opacity-40' : ''}"
				title={describe(bucket)}
			></div>
		{:else}
			<div class="h-full w-full rounded-fc-xs bg-fc-border opacity-40"></div>
		{/each}
	</div>

	{#if buckets.length > 0}
		<div class="flex items-center justify-between font-fc-mono text-fc-xs text-fc-fg-muted">
			<span>{covered}% covered</span>
			<span>{outage > 0 ? `${duration(outage)} down` : 'no downtime'}</span>
		</div>
	{/if}

	<div class="sr-only">
		<table>
			<caption>{label}</caption>
			<thead>
				<tr><th scope="col">Bucket</th><th scope="col">Uptime</th><th scope="col">Checks</th></tr>
			</thead>
			<tbody>
				{#each buckets as bucket (bucket.bucket_start)}
					<tr>
						<td>{new Date(bucket.bucket_start).toLocaleString()}</td>
						<td>{bucket.total === 0 ? 'not probed' : percent(bucket.uptime, 1)}</td>
						<td>{bucket.total}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
</div>
