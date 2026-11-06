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
	import { dateTime, duration, percent } from '$lib/format';
	import { t } from '$lib/i18n/index.svelte';

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
			error = err instanceof Error ? err.message : t('monitor.loadFailed');
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
			toast.success(t('monitor.saved'));
		} catch (err) {
			formError = err instanceof Error ? err.message : t('monitor.saveFailed');
		} finally {
			busy = false;
		}
	}

	async function confirmDelete() {
		try {
			await backend.deleteMonitor(monitorId);
			toast.success(t('monitors.deleted', { name: monitor?.name ?? t('monitor.title') }));
			goto('/monitors');
		} catch (err) {
			/* `error` is the load failure, and it has no dismissal. A failed delete leaves the
			   monitor on screen and intact, so the toast is the right and only report. */
			toast.danger(err instanceof Error ? err.message : t('monitors.deleteFailed'));
			throw err;
		}
	}

	function incidentLength(incident: Incident): string {
		const end = incident.resolved_at ? new Date(incident.resolved_at).getTime() : Date.now();
		return duration((end - new Date(incident.opened_at).getTime()) / 1000);
	}

	const windowKeys = {
		'24h': 'monitor.window.24h',
		'7d': 'monitor.window.7d',
		'30d': 'monitor.window.30d',
		'90d': 'monitor.window.90d'
	} as const;
</script>

<svelte:head>
	<title>{monitor ? monitor.name : t('monitor.title')} · Sonde</title>
</svelte:head>

{#if error && !monitor}
	<Alert tone="danger" title={t('monitor.loadError')}>{error}</Alert>
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
			description={monitor.type === 'push' ? t('monitor.pushHeartbeat') : monitor.target}
			back={{ href: '/monitors', label: t('nav.monitors') }}
		>
			{#snippet actions()}
				{@const current = monitor!}
				<Badge tone="neutral">{current.type}</Badge>
				<StatusDot
					tone={current.enabled ? 'success' : 'neutral'}
					label={current.enabled ? t('monitor.watching') : t('monitor.status.paused')}
				/>
				<Button variant="danger" icon={icons.remove} onclick={() => (confirmOpen = true)}>
					{t('monitor.delete')}
				</Button>
			{/snippet}
		</PageHeader>

		{#if error}
			<Alert tone="danger" title={t('monitor.somethingWrong')}>{error}</Alert>
		{/if}

		<div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
			{#each uptime as stat (stat.window)}
				<StatCard
					label={stat.window in windowKeys ? t(windowKeys[stat.window as keyof typeof windowKeys]) : stat.window}
					value={percent(stat.uptime)}
					delta={stat.total === 0
						? t('monitor.noChecks')
						: t('monitor.failedOf', { failed: stat.failed, total: stat.total })}
				/>
			{/each}
		</div>

		<MonitorHistoryPanel {monitorId} onError={(message) => (error = message)} />

		{#if heartbeatUrl}
			<!-- POST only. internal/checker/push.go registers no GET, and a GET falls through to
			     the SPA catch-all and answers 200 with HTML, so a curl line without -X POST
			     reports success while the monitor goes down. -->
			<Section
				title={t('monitor.heartbeat.title')}
				description={t('monitor.heartbeat.description')}
			>
				<div class="flex flex-col gap-4">
					<SecretField label={t('monitor.heartbeat.url')} value={heartbeatUrl} />
					<SecretField label={t('monitor.heartbeat.cron')} value={heartbeatCommand} />
				</div>
			</Section>
		{/if}

		<div class="grid grid-cols-1 gap-10 lg:grid-cols-3">
			<Section title={t('monitor.settings')} class="lg:col-span-2">
				<Card>
					<form onsubmit={save} class="flex flex-col gap-4">
						{#if formError}
							<Alert tone="danger" title={t('monitor.notSaved')}>{formError}</Alert>
						{/if}
						<MonitorForm {value} {busy} />
					</form>
				</Card>
			</Section>

			<Section title={t('nav.incidents')}>
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
						title={t('incidents.empty.title')}
						description={t('monitor.noIncidents')}
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
									<span>{incident.resolved_at ? t('monitor.incident.closed') : t('monitor.incident.open')}</span>
									<span>·</span>
									<span>{dateTime(incident.opened_at)}</span>
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
	title={monitor
		? t('monitors.delete.title', { name: monitor.name })
		: t('monitors.delete.titleFallback')}
	description={t('monitors.delete.description')}
	confirmLabel={t('monitors.delete.confirm')}
	onConfirm={confirmDelete}
/>
