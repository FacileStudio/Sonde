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
	import { milliseconds, percent, since, statusLabel, STATUS_TONE } from '$lib/format';
	import { t } from '$lib/i18n/index.svelte';

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
			error = err instanceof Error ? err.message : t('monitors.loadFailed');
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
		toast.success(t('monitors.created', { name: monitor.name }));
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
			toast.success(t('monitors.deleted', { name: monitor.name }));
			await load();
		} catch (err) {
			/* The toast is the whole report. `error` means "the list never loaded" and is the
			   first render branch, so writing a failed delete into it replaces the table the
			   user is still looking at with a load error that did not happen. */
			toast.danger(err instanceof Error ? err.message : t('monitors.deleteFailed'));
			throw err;
		}
	}
</script>

<svelte:head>
	<title>{t('nav.monitors')} · Sonde</title>
</svelte:head>

<div class="flex flex-col gap-10">
	<PageHeader title={t('nav.monitors')} description={t('monitors.description')}>
		{#snippet actions()}
			<Button icon={icons.plus} onclick={() => (creating = true)}>{t('monitors.new')}</Button>
		{/snippet}
	</PageHeader>

	{#if error}
		<Alert tone="danger" title={t('monitors.loadError')}>{error}</Alert>
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
			title={t('monitors.empty.title')}
			description={t('monitors.empty.description')}
		>
			<Button variant="secondary" icon={icons.plus} onclick={() => (creating = true)}>
				{t('monitors.empty.cta')}
			</Button>
		</EmptyState>
	{:else if totals}
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
			<StatCard
				label={t('monitors.stat.up')}
				value={totals.up}
				delta={t('monitors.stat.ofMonitors', { count: totals.monitors })}
			/>
			<StatCard
				label={t('monitors.stat.down')}
				value={totals.down}
				delta={totals.open_incidents > 0
					? t('monitors.stat.openIncidents', { count: totals.open_incidents })
					: t('monitors.stat.noOpenIncidents')}
			/>
			<StatCard
				label={t('monitors.stat.uptime24')}
				value={percent(totals.uptime_24h)}
				delta={t('monitors.stat.checks', { count: totals.checks_24h })}
			/>
			<StatCard label={t('monitors.stat.latency24')} value={milliseconds(totals.avg_latency_24h)}>
				<Sparkline data={latencySeries} height={28} emptyLabel={t('monitors.stat.noProbes')} />
			</StatCard>
		</div>

		<Table>
			<thead>
				<tr>
					<th scope="col">{t('monitors.col.monitor')}</th>
					<th scope="col" class="w-48">{t('monitors.col.last24')}</th>
					<th scope="col">{t('monitors.col.uptime')}</th>
					<th scope="col">{t('monitors.col.latency')}</th>
					<th scope="col">{t('monitors.col.checked')}</th>
					<th scope="col"><span class="sr-only">{t('common.actions')}</span></th>
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
									label={statusLabel(monitor.status)}
								/>
								<span
									class="truncate font-medium text-fc-fg transition-colors group-hover:text-fc-accent"
								>
									{monitor.name}
								</span>
								<Badge tone="neutral">{monitor.type}</Badge>
								{#if monitor.open_incidents > 0}
									<Badge tone="danger">
										{t('monitors.openBadge', { count: monitor.open_incidents })}
									</Badge>
								{/if}
							</div>
						</td>
						<td class="w-48">
							<UptimeBar
								buckets={monitor.history}
								height={16}
								label={t('monitors.uptimeLabel', { name: monitor.name })}
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
									aria-label={t('monitors.openAria', { name: monitor.name })}
									href={`/monitors/${monitor.id}`}
								/>
								<Button
									variant="ghost-danger"
									size="sm"
									icon={icons.remove}
									aria-label={t('monitors.deleteAria', { name: monitor.name })}
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
	title={pendingDelete
		? t('monitors.delete.title', { name: pendingDelete.name })
		: t('monitors.delete.titleFallback')}
	description={t('monitors.delete.description')}
	confirmLabel={t('monitors.delete.confirm')}
	onConfirm={confirmDelete}
	onCancel={() => (pendingDelete = null)}
/>
