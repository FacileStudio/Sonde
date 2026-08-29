<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import {
		Alert,
		Badge,
		Button,
		Card,
		ConfirmModal,
		EmptyState,
		PageHeader,
		Section,
		SecretField,
		Skeleton,
		StatCard,
		StatusDot,
		icons,
		toast
	} from '@facile/muse';
	import MonitorForm from '$lib/components/MonitorForm.svelte';
	import MonitorHistoryPanel from '$lib/components/MonitorHistory.svelte';
	import {
		backend,
		type Incident,
		type Monitor,
		type MonitorInput,
		type UptimeResponse
	} from '$lib/backend';
	import { duration, percent } from '$lib/format';

	let monitorId = $derived(Number(page.params.id));

	let monitor = $state<Monitor | null>(null);
	let incidents = $state<Incident[]>([]);
	let uptime = $state<UptimeResponse[]>([]);
	let value = $state<MonitorInput | null>(null);
	let loading = $state(true);
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
			[incidents, uptime] = await Promise.all([
				backend.monitorIncidents(monitorId),
				backend.monitorUptimeSummary(monitorId)
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

		<MonitorHistoryPanel {monitorId} onError={(message) => (error = message)} />

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
