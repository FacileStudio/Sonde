<script lang="ts">
	import { Sparkline } from '@facile/muse';
	import type { Check } from '$lib/backend';

	let {
		checks = [],
		slots = 30,
		height = 20,
		showSparkline = false,
		showLatency = false
	}: {
		checks?: Check[];
		slots?: number;
		height?: number;
		showSparkline?: boolean;
		showLatency?: boolean;
	} = $props();

	const padded = $derived.by(() => {
		const recent = checks.slice(-slots);
		const emptyCount = Math.max(0, slots - recent.length);
		const empty: (Check | null)[] = Array(emptyCount).fill(null);
		return [...empty, ...recent];
	});

	const latencies = $derived(
		checks.map((c) => (c.ok ? c.latency_ms : 0))
	);

	const lastCheck = $derived(checks.length > 0 ? checks[checks.length - 1] : null);
	const avgLatency = $derived.by(() => {
		const okChecks = checks.filter((c) => c.ok);
		if (okChecks.length === 0) return null;
		const sum = okChecks.reduce((acc, c) => acc + c.latency_ms, 0);
		return Math.round(sum / okChecks.length);
	});
</script>

<div class="flex flex-col gap-1.5 min-w-0">
	<div class="flex items-center gap-0.5" style="height: {height}px;">
		{#each padded as item, index (item ? item.id : `empty-${index}`)}
			{#if item}
				<div
					class="flex-1 h-full min-w-[3px] max-w-[6px] rounded-fc-xs transition-transform hover:scale-125 hover:z-10 {item.ok ? 'bg-emerald-500 dark:bg-emerald-400' : 'bg-rose-500 dark:bg-rose-400'}"
					title={`${new Date(item.created_at).toLocaleTimeString()}: ${item.ok ? `${item.latency_ms}ms` : item.error || 'Down'}`}
				></div>
			{:else}
				<div
					class="flex-1 h-full min-w-[3px] max-w-[6px] rounded-fc-xs bg-fc-border-subtle opacity-40"
					title="No probe data"
				></div>
			{/if}
		{/each}
	</div>

	{#if showLatency && lastCheck}
		<div class="flex items-center justify-between text-fc-xs text-fc-fg-muted font-fc-mono">
			<span>{lastCheck.ok ? `${lastCheck.latency_ms}ms current` : 'Down'}</span>
			{#if avgLatency !== null}
				<span>avg {avgLatency}ms</span>
			{/if}
		</div>
	{/if}

	{#if showSparkline && latencies.length > 1}
		<div class="mt-2">
			<Sparkline
				data={latencies}
				height={40}
				color="var(--color-fc-chart-1)"
				valueFormat={(n) => `${n}ms`}
			/>
		</div>
	{/if}
</div>
