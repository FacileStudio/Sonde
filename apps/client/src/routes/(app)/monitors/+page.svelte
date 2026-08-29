<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import {
		Alert,
		Badge,
		Button,
		ConfirmModal,
		EmptyState,
		PageHeader,
		Skeleton,
		Sparkline,
		StatCard,
		StatusDot,
		Table,
		icons,
		toast
	} from '@facile/muse';
	import MonitorDrawer from '$lib/components/MonitorDrawer.svelte';
	import UptimeBar from '$lib/components/UptimeBar.svelte';
	import { backend, type Fleet, type MonitorSummary } from '$lib/backend';
	import { milliseconds, percent, since, STATUS_LABEL, STATUS_TONE } from '$lib/format';

	let fleet = $state<Fleet | null>(null);
	let loading = $state(true);
	let error = $state('');
	let creating = $state(false);
	let pendingDelete = $state<MonitorSummary | null>(null);
	let confirmOpen = $state(false);

	onMount(load);

	async function load() {
		try {
			fleet = await backend.monitorsOverview();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load monitors.';
		} finally {
			loading = false;
		}
	}

	const totals = $derived(fleet?.totals ?? null);
	const monitors = $derived(fleet?.monitors ?? []);

	/* Averaged across monitors, hour by hour, over the buckets that were actually
	   probed. An hour nobody measured carries the previous hour forward rather
	   than dropping to zero, which would draw an outage that never happened. */
	const latencySeries = $derived.by(() => {
		const width = monitors.reduce((longest, monitor) => Math.max(longest, monitor.history.length), 0);
		let last = 0;
		return Array.from({ length: width }, (_, index) => {
			let sum = 0;
			let count = 0;
			for (const monitor of monitors) {
				const bucket = monitor.history[index];
				if (bucket && bucket.total > 0) {
					sum += bucket.avg_latency_ms;
					count += 1;
				}
			}
			if (count > 0) last = sum / count;
			return last;
		});
	});

	async function onCreated(monitor: { name: string }) {
		toast.success(`${monitor.name} is being watched.`);
		await load();
	}

	function askDelete(monitor: MonitorSummary) {
		pendingDelete = monitor;
		confirmOpen = true;
	}

	async function confirmDelete() {
		const monitor = pendingDelete;
		if (!monitor) return;
		try {
			await backend.deleteMonitor(monitor.id);
			pendingDelete = null;
			toast.success(`${monitor.name} deleted.`);
			await load();
		} catch (err) {
			/* The toast is the whole report. `error` means "the list never loaded" and is the
			   first render branch, so writing a failed delete into it replaces the table the
			   user is still looking at with a load error that did not happen. */
			toast.danger(err instanceof Error ? err.message : 'Could not delete the monitor.');
			throw err;
		}
	}
</script>

<svelte:head>
	<title>Monitors — Sonde</title>
</svelte:head>

<div class="flex flex-col gap-10">
	<PageHeader title="Monitors" description="What Sonde watches, and whether it is alive right now.">
		{#snippet actions()}
			<Button icon={icons.plus} onclick={() => (creating = true)}>New monitor</Button>
		{/snippet}
	</PageHeader>

	{#if error}
		<Alert tone="danger" title="Could not load monitors">{error}</Alert>
	{:else if loading}
		<div class="flex flex-col gap-4">
			<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
				<Skeleton class="h-24 w-full" />
				<Skeleton class="h-24 w-full" />
				<Skeleton class="h-24 w-full" />
				<Skeleton class="h-24 w-full" />
			</div>
			<Skeleton class="h-10 w-full" />
			<Skeleton class="h-10 w-full" />
			<Skeleton class="h-10 w-full" />
		</div>
	{:else if monitors.length === 0}
		<EmptyState
			icon={icons.monitor}
			title="No monitors yet"
			description="Add an HTTP endpoint, a TCP port, or a push heartbeat to start watching."
		>
			<Button variant="outline" icon={icons.plus} onclick={() => (creating = true)}>
				Create your first monitor
			</Button>
		</EmptyState>
	{:else if totals}
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
			<StatCard label="Up" value={totals.up} delta={`of ${totals.monitors} monitors`} />
			<StatCard
				label="Down"
				value={totals.down}
				delta={totals.open_incidents > 0
					? `${totals.open_incidents} open incidents`
					: 'no open incidents'}
			/>
			<StatCard
				label="Uptime 24 h"
				value={percent(totals.uptime_24h)}
				delta={`${totals.checks_24h} checks`}
			/>
			<StatCard label="Avg latency 24 h" value={milliseconds(totals.avg_latency_24h)}>
				<Sparkline data={latencySeries} height={28} emptyLabel="No probes yet" />
			</StatCard>
		</div>

		<Table>
			<thead>
				<tr>
					<th scope="col">Monitor</th>
					<th scope="col" class="w-48">Last 24 h</th>
					<th scope="col">Uptime</th>
					<th scope="col">Latency</th>
					<th scope="col">Checked</th>
					<th scope="col"><span class="sr-only">Actions</span></th>
				</tr>
			</thead>
			<tbody>
				{#each monitors as monitor (monitor.id)}
					<tr
						class="group cursor-pointer transition-colors hover:bg-fc-surface"
						onclick={(e) => {
							const target = e.target as HTMLElement;
							if (target.closest('button, a')) return;
							goto(`/monitors/${monitor.id}`);
						}}
					>
						<td>
							<div class="flex min-w-0 items-center gap-2">
								<StatusDot
									tone={STATUS_TONE[monitor.status]}
									label={STATUS_LABEL[monitor.status]}
								/>
								<span
									class="truncate font-medium text-fc-fg transition-colors group-hover:text-fc-accent"
								>
									{monitor.name}
								</span>
								<Badge tone="neutral">{monitor.type}</Badge>
								{#if monitor.open_incidents > 0}
									<Badge tone="danger">
										{monitor.open_incidents} open
									</Badge>
								{/if}
							</div>
						</td>
						<td class="w-48">
							<UptimeBar
								buckets={monitor.history}
								height={16}
								label={`${monitor.name} uptime, last 24 hours`}
							/>
						</td>
						<td class="font-fc-mono text-fc-sm text-fc-fg-muted">
							{percent(monitor.uptime_24h)}
						</td>
						<td class="font-fc-mono text-fc-sm text-fc-fg-muted">
							{milliseconds(monitor.last_latency_ms)}
						</td>
						<td class="text-fc-sm text-fc-fg-muted">{since(monitor.last_check_at)}</td>
						<td>
							<div class="flex justify-end gap-1">
								<Button
									variant="ghost"
									size="sm"
									icon={icons.edit}
									aria-label={`Open ${monitor.name}`}
									href={`/monitors/${monitor.id}`}
								/>
								<Button
									variant="ghost-danger"
									size="sm"
									icon={icons.remove}
									aria-label={`Delete ${monitor.name}`}
									onclick={() => askDelete(monitor)}
								/>
							</div>
						</td>
					</tr>
				{/each}
			</tbody>
		</Table>
	{/if}
</div>

<MonitorDrawer bind:open={creating} {onCreated} />

<ConfirmModal
	bind:open={confirmOpen}
	tone="danger"
	title={pendingDelete ? `Delete ${pendingDelete.name}?` : 'Delete monitor?'}
	description="Its check history, uptime record and incidents are deleted with it. This cannot be undone."
	confirmLabel="Delete monitor"
	onConfirm={confirmDelete}
	onCancel={() => (pendingDelete = null)}
/>
