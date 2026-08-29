<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import {
		Alert,
		Badge,
		BarChart,
		Button,
		Card,
		ConfirmModal,
		EmptyState,
		LineChart,
		PageHeader,
		Section,
		SecretField,
		Skeleton,
		StatCard,
		StatusDot,
		Tabs,
		chartColor,
		icons,
		toast
	} from '@facile/muse';
	import MonitorForm from '$lib/components/MonitorForm.svelte';
	import UptimeBar from '$lib/components/UptimeBar.svelte';
	import {
		backend,
		UPTIME_WINDOWS,
		type Incident,
		type Monitor,
		type MonitorHistory,
		type MonitorInput,
		type UptimeResponse,
		type UptimeWindow
	} from '$lib/backend';
	import { duration, latencySeries, milliseconds, percent, shortDay, shortTime } from '$lib/format';

	let monitorId = $derived(Number(page.params.id));

	let monitor = $state<Monitor | null>(null);
	let incidents = $state<Incident[]>([]);
	let uptime = $state<UptimeResponse[]>([]);
	let history = $state<MonitorHistory | null>(null);
	let value = $state<MonitorInput | null>(null);
	let range = $state<UptimeWindow>('24h');
	let loading = $state(true);
	let historyLoading = $state(false);
	let busy = $state(false);
	let error = $state('');
	let formError = $state('');
	let confirmOpen = $state(false);

	const heartbeatUrl = $derived(
		monitor?.type === 'push' && monitor.push_token
			? `${window.location.origin}/api/push/${monitor.push_token}`
			: ''
	);

	/* -f so a refused heartbeat is a non-zero exit the cron daemon reports, -sS so it stays
	   quiet until it fails. */
	const heartbeatCommand = $derived(heartbeatUrl ? `curl -fsS -X POST ${heartbeatUrl}` : '');

	const rangeTabs = UPTIME_WINDOWS.map((entry) => ({ id: entry, label: entry }));

	const buckets = $derived(history?.buckets ?? []);
	const resolution = $derived(history?.resolution ?? 'hour');
	const labels = $derived(
		buckets.map((bucket) =>
			resolution === 'day' ? shortDay(bucket.bucket_start) : shortTime(bucket.bucket_start)
		)
	);
	const latency = $derived([
		{ name: 'Average latency', data: latencySeries(buckets), color: chartColor(0) }
	]);
	/* One "failed checks" series, not failed stacked on successful: a healthy monitor
	   reports ~60 successes to a single failure per bucket, and the failed segment of that
	   stack is under a pixel — the one thing the chart exists to show. */
	const failures = $derived([
		{
			name: 'Failed checks',
			data: buckets.map((bucket) => bucket.failed),
			color: chartColor(3)
		}
	]);

	async function loadHistory(window: UptimeWindow) {
		historyLoading = true;
		try {
			history = await backend.monitorHistory(monitorId, window);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load the history.';
		} finally {
			historyLoading = false;
		}
	}

	function selectRange(id: string) {
		range = id as UptimeWindow;
		void loadHistory(range);
	}

	onMount(async () => {
		try {
			const loaded = await backend.getMonitor(monitorId);
			monitor = loaded;
			value = {
				slug: loaded.slug,
				name: loaded.name,
				type: loaded.type,
				target: loaded.target,
				interval_seconds: loaded.interval_seconds,
				timeout_seconds: loaded.timeout_seconds,
				expected_status: loaded.expected_status,
				expected_keyword: loaded.expected_keyword,
				enabled: loaded.enabled
			};
			[incidents, uptime, history] = await Promise.all([
				backend.monitorIncidents(monitorId),
				backend.monitorUptimeSummary(monitorId),
				backend.monitorHistory(monitorId, range)
			]);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load monitor.';
		} finally {
			loading = false;
		}
	});

	async function save(event: SubmitEvent) {
		event.preventDefault();
		if (!value || busy) return;
		busy = true;
		formError = '';
		try {
			monitor = await backend.updateMonitor(monitorId, value);
			toast.success('Monitor saved.');
		} catch (err) {
			formError = err instanceof Error ? err.message : 'Failed to save the monitor.';
		} finally {
			busy = false;
		}
	}

	async function confirmDelete() {
		try {
			await backend.deleteMonitor(monitorId);
			toast.success(`${monitor?.name ?? 'Monitor'} deleted.`);
			goto('/monitors');
		} catch (err) {
			/* `error` is the load failure, and it has no dismissal. A failed delete leaves the
			   monitor on screen and intact, so the toast is the right and only report. */
			toast.danger(err instanceof Error ? err.message : 'Could not delete the monitor.');
			throw err;
		}
	}

	function incidentLength(incident: Incident): string {
		const end = incident.resolved_at ? new Date(incident.resolved_at).getTime() : Date.now();
		return duration((end - new Date(incident.opened_at).getTime()) / 1000);
	}

	const windowLabel: Record<string, string> = {
		'24h': 'Uptime, 24 hours',
		'7d': 'Uptime, 7 days',
		'30d': 'Uptime, 30 days',
		'90d': 'Uptime, 90 days'
	};
</script>

<svelte:head>
	<title>{monitor ? `${monitor.name} — Sonde` : 'Monitor — Sonde'}</title>
</svelte:head>

{#if error && !monitor}
	<Alert tone="danger" title="Could not load this monitor">{error}</Alert>
{:else if loading || !monitor || !value}
	<div class="flex flex-col gap-4">
		<Skeleton class="h-10 w-64" />
		<Skeleton class="h-24 w-full" />
		<Skeleton class="h-64 w-full" />
	</div>
{:else}
	<div class="flex flex-col gap-10">
		<PageHeader
			title={monitor.name}
			description={monitor.type === 'push' ? 'Push heartbeat' : monitor.target}
			back={{ href: '/monitors', label: 'Monitors' }}
		>
			{#snippet actions()}
				{@const current = monitor!}
				<Badge tone="neutral">{current.type}</Badge>
				<StatusDot
					tone={current.enabled ? 'success' : 'neutral'}
					label={current.enabled ? 'Watching' : 'Paused'}
				/>
				<Button variant="danger" icon={icons.remove} onclick={() => (confirmOpen = true)}>
					Delete
				</Button>
			{/snippet}
		</PageHeader>

		{#if error}
			<Alert tone="danger" title="Something went wrong">{error}</Alert>
		{/if}

		<div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
			{#each uptime as stat (stat.window)}
				<StatCard
					label={windowLabel[stat.window] ?? stat.window}
					value={percent(stat.uptime)}
					delta={stat.total === 0
						? 'No checks recorded yet'
						: `${stat.failed} failed of ${stat.total} checks`}
				/>
			{/each}
		</div>

		<Section title="History" description="Coverage, latency and failures over the chosen window.">
			{#snippet actions()}
				<Tabs items={rangeTabs} value={range} onChange={selectRange} label="History window" />
			{/snippet}

			{#if historyLoading}
				<Skeleton class="h-64 w-full" />
			{:else}
				<div class="flex flex-col gap-4">
					<Card class="p-5">
						<UptimeBar {buckets} {resolution} height={32} label="Uptime history" />
					</Card>

					<div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
						<Card class="flex flex-col gap-4 p-5">
							<div>
								<h3 class="text-fc-md font-semibold text-fc-fg">Response time</h3>
								<p class="text-fc-xs text-fc-fg-muted">
									Average latency per {resolution}
								</p>
							</div>
							<LineChart
								class="flex-1"
								series={latency}
								{labels}
								area
								height={220}
								yFormat={(n) => milliseconds(n)}
								emptyLabel="No checks in this window"
							/>
						</Card>

						<Card class="flex flex-col gap-4 p-5">
							<div>
								<h3 class="text-fc-md font-semibold text-fc-fg">Failed checks</h3>
								<p class="text-fc-xs text-fc-fg-muted">
									Probes that did not pass, per {resolution}
								</p>
							</div>
							<BarChart
								class="flex-1"
								series={failures}
								{labels}
								height={220}
								emptyLabel="No checks in this window"
							/>
						</Card>
					</div>
				</div>
			{/if}
		</Section>

		{#if heartbeatUrl}
			<!-- POST only. internal/checker/push.go registers no GET, and a GET falls through to
			     the SPA catch-all and answers 200 with HTML, so a curl line without -X POST
			     reports success while the monitor goes down. -->
			<Section
				title="Heartbeat"
				description="Have the job POST this URL every interval. A GET answers 200 without recording anything, so the method matters. Silence past the interval opens an incident."
			>
				<div class="flex flex-col gap-4">
					<SecretField label="Heartbeat URL" value={heartbeatUrl} />
					<SecretField label="Cron command" value={heartbeatCommand} />
				</div>
			</Section>
		{/if}

		<div class="grid grid-cols-1 gap-10 lg:grid-cols-3">
			<Section title="Settings" class="lg:col-span-2">
				<Card>
					<form onsubmit={save} class="flex flex-col gap-4">
						{#if formError}
							<Alert tone="danger" title="Not saved">{formError}</Alert>
						{/if}
						<MonitorForm {value} {busy} />
					</form>
				</Card>
			</Section>

			<Section title="Incidents">
				{#snippet actions()}
					{#if incidents.length > 0}
						<Badge tone={incidents.some((entry) => !entry.resolved_at) ? 'danger' : 'neutral'}>
							{incidents.length}
						</Badge>
					{/if}
				{/snippet}
				{#if incidents.length === 0}
					<EmptyState
						icon={icons.shield}
						title="No incidents"
						description="Nothing has gone wrong on this monitor yet."
					/>
				{:else}
					<ul class="flex flex-col gap-3">
						{#each incidents as incident (incident.id)}
							<li
								class="flex min-w-0 flex-col gap-1 border-l-2 pl-3 {incident.resolved_at
									? 'border-fc-border'
									: 'border-fc-danger'}"
							>
								<span class="text-fc-sm text-fc-fg">{incident.cause || '—'}</span>
								<span class="flex flex-wrap items-center gap-2 text-fc-xs text-fc-fg-muted">
									<span class="font-fc-mono">{incidentLength(incident)}</span>
									<span>{incident.resolved_at ? 'down' : 'down, still open'}</span>
									<span>·</span>
									<span>{new Date(incident.opened_at).toLocaleString()}</span>
								</span>
							</li>
						{/each}
					</ul>
				{/if}
			</Section>
		</div>
	</div>
{/if}

<ConfirmModal
	bind:open={confirmOpen}
	tone="danger"
	title={monitor ? `Delete ${monitor.name}?` : 'Delete monitor?'}
	description="Its check history, uptime record and incidents are deleted with it. This cannot be undone."
	confirmLabel="Delete monitor"
	onConfirm={confirmDelete}
/>
