<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { Badge, Button, Card, PageHeader, Section, Spinner, StatusDot } from '@facile/muse';
	import MonitorForm from '$lib/components/MonitorForm.svelte';
	import { backend, type Incident, type Monitor, type MonitorInput, type UptimeResponse } from '$lib/backend';

	let monitorId = $derived(Number(page.params.id));

	let monitor = $state<Monitor | null>(null);
	let incidents = $state<Incident[]>([]);
	let uptime24h = $state<UptimeResponse | null>(null);
	let value = $state<MonitorInput | null>(null);
	let busy = $state(false);
	let error = $state('');
	let saved = $state(false);

	onMount(async () => {
		try {
			monitor = await backend.getMonitor(monitorId);
			value = {
				slug: monitor.slug,
				name: monitor.name,
				type: monitor.type,
				target: monitor.target,
				interval_seconds: monitor.interval_seconds,
				timeout_seconds: monitor.timeout_seconds,
				expected_status: monitor.expected_status,
				expected_keyword: monitor.expected_keyword,
				enabled: monitor.enabled
			};
			[incidents, uptime24h] = await Promise.all([
				backend.monitorIncidents(monitorId),
				backend.monitorUptime(monitorId, '24h').catch(() => null)
			]);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load monitor.';
		}
	});

	async function save(event: SubmitEvent) {
		event.preventDefault();
		if (!value || busy) return;
		busy = true;
		error = '';
		saved = false;
		try {
			await backend.updateMonitor(monitorId, value);
			saved = true;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to save monitor.';
		} finally {
			busy = false;
		}
	}

	async function remove() {
		if (!confirm(`Delete monitor "${monitor?.name}"? Its history goes with it.`)) return;
		await backend.deleteMonitor(monitorId).catch(() => undefined);
		goto('/');
	}
</script>

<svelte:head>
	<title>{monitor ? `${monitor.name} — Sonde` : 'Monitor — Sonde'}</title>
</svelte:head>

{#if !monitor || !value}
	<div class="flex justify-center py-16"><Spinner /></div>
{:else}
	<PageHeader
		title={monitor.name}
		description={monitor.target || 'Push heartbeat'}
		back={{ href: '/', label: 'Monitors' }}
	>
		{#snippet actions()}
			<Button variant="danger" onclick={remove}>Delete</Button>
		{/snippet}
	</PageHeader>

	<div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
		<Card class="p-6 lg:col-span-2">
			<form onsubmit={save}>
				<MonitorForm {value} submitLabel="Save changes" {busy} {error} />
			</form>
			{#if saved}
				<div class="mt-3 text-fc-sm text-fc-success">Saved.</div>
			{/if}
		</Card>

		<div class="flex flex-col gap-6">
			<Section title="Status">
				<div class="flex items-center gap-2">
					<StatusDot tone={monitor.enabled ? 'success' : 'neutral'} pulse={monitor.enabled} />
					<span class="text-fc-sm text-fc-fg">{monitor.enabled ? 'Watching' : 'Paused'}</span>
					{#if uptime24h}
						<Badge tone="info">{uptime24h.uptime === null ? 'no data' : `${uptime24h.uptime.toFixed(2)}% / 24h`}</Badge>
					{/if}
				</div>
				{#if monitor.type === 'push' && monitor.push_token}
					<div class="mt-3 text-fc-xs text-fc-fg-muted">
						Heartbeat URL
						<code class="block truncate rounded border border-fc-border bg-fc-surface px-2 py-1 font-mono">
							{`${window.location.origin}/api/push/${monitor.push_token}`}
						</code>
					</div>
				{/if}
			</Section>

			<Section title="Incidents">
				{#if incidents.length === 0}
					<p class="text-fc-sm text-fc-fg-muted">Nothing has gone wrong yet.</p>
				{:else}
					<ul class="flex flex-col gap-2">
						{#each incidents as incident (incident.id)}
							<li class="flex flex-col gap-0.5 border-l-2 pl-3 {incident.resolved_at
								? 'border-fc-border'
								: 'border-fc-danger'}">
								<span class="text-fc-sm text-fc-fg">{incident.cause}</span>
								<span class="text-fc-xs text-fc-fg-muted">
									{new Date(incident.opened_at).toLocaleString()}
									{incident.resolved_at ? `— resolved ${new Date(incident.resolved_at).toLocaleTimeString()}` : '— ongoing'}
								</span>
							</li>
						{/each}
					</ul>
				{/if}
			</Section>
		</div>
	</div>
{/if}
