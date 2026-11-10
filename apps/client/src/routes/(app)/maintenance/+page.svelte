<script lang="ts">
	import { onMount } from 'svelte';
	import { Alert, EmptyState, PageHeader, Skeleton, StatCard, Tabs, icons } from '@facile/muse';
	import MaintenanceActions from '$lib/components/MaintenanceActions.svelte';
	import MaintenanceConformance from '$lib/components/MaintenanceConformance.svelte';
	import MaintenanceDrift from '$lib/components/MaintenanceDrift.svelte';
	import MaintenanceGraph from '$lib/components/MaintenanceGraph.svelte';
	import MaintenanceProjects from '$lib/components/MaintenanceProjects.svelte';
	import {
		backend,
		type MaintenanceActions as Actions,
		type MaintenanceConformance as Conformance,
		type MaintenanceDrift as Drift,
		type MaintenanceGraph as Graph,
		type MaintenanceProject,
		type MaintenanceRun
	} from '$lib/backend';
	import { since } from '$lib/format';
	import { t } from '$lib/i18n/index.svelte';

	const staleAfterHours = 26;

	let runs = $state<MaintenanceRun[]>([]);
	let actions = $state<Actions | null>(null);
	let drift = $state<Drift | null>(null);
	let projects = $state<MaintenanceProject[]>([]);
	let graph = $state<Graph>({ nodes: [], edges: [] });
	let conformance = $state<Conformance>({ checks: [], projects: [] });
	let loading = $state(true);
	let error = $state('');
	let tab = $state('actions');

	const panelId = 'maintenance-panel';

	const tabs = $derived([
		{ id: 'actions', label: t('maintenance.tab.actions') },
		{ id: 'drift', label: t('maintenance.tab.drift') },
		{ id: 'graph', label: t('maintenance.tab.graph') },
		{ id: 'conformance', label: t('maintenance.tab.conformance') },
		{ id: 'projects', label: t('maintenance.tab.projects') }
	]);

	const activeLabel = $derived(tabs.find((item) => item.id === tab)?.label ?? '');

	/* A failed refresh after a successful save keeps the previous ranking: the role is stored, and
	   replacing the whole page with an error would hide what was already on screen. */
	async function refreshActions() {
		actions = await backend.maintenanceActions().catch(() => actions);
	}

	const last = $derived(runs[0]);
	const stale = $derived(
		last ? Date.now() - new Date(last.finished_at).getTime() > staleAfterHours * 3600 * 1000 : false
	);

	onMount(async () => {
		try {
			[runs, actions, drift, projects, graph, conformance] = await Promise.all([
				backend.maintenanceRuns(),
				backend.maintenanceActions(),
				backend.maintenanceDrift(),
				backend.maintenanceProjects(),
				backend.maintenanceGraph(),
				backend.maintenanceConformance()
			]);
		} catch (err) {
			const status = (err as { status?: number }).status;
			error = status === 403 ? t('maintenance.forbidden') : err instanceof Error ? err.message : t('maintenance.loadError');
		} finally {
			loading = false;
		}
	});
</script>

<svelte:head>
	<title>{t('nav.maintenance')} · Sonde</title>
</svelte:head>

<div class="flex flex-col gap-10">
	<PageHeader title={t('nav.maintenance')} description={t('maintenance.description')} />

	{#if error}
		<Alert tone="danger" title={t('maintenance.loadError')}>{error}</Alert>
	{:else if loading}
		<div class="flex flex-col gap-4">
			<Skeleton class="h-24 w-full" />
			<Skeleton class="h-64 w-full" />
		</div>
	{:else if !last || !actions || !drift}
		<EmptyState icon={icons.shield} title={t('maintenance.empty.title')} description={t('maintenance.empty.description')} />
	{:else}
		{#if stale}
			<Alert tone="warning" title={t('maintenance.stale.title')}>
				{t('maintenance.stale.body', { when: since(last.finished_at) })}
			</Alert>
		{/if}
		{#if !last.complete}
			<Alert tone="warning" title={t('maintenance.incomplete.title')}>
				{t('maintenance.incomplete.body', { scanned: last.repos_scanned, expected: last.repos_expected })}
			</Alert>
		{/if}

		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
			<StatCard label={t('maintenance.stat.act')} value={actions.counts.act} delta={t('maintenance.stat.actHint')} />
			<StatCard
				label={t('maintenance.stat.schedule')}
				value={actions.counts.schedule}
				delta={t('maintenance.stat.scheduleHint')}
			/>
			<StatCard
				label={t('maintenance.stat.open')}
				value={last.findings_open}
				delta={t('maintenance.stat.openHint', { count: actions.actions.length })}
			/>
			<StatCard
				label={t('maintenance.stat.lastRun')}
				value={since(last.finished_at)}
				delta={t('maintenance.stat.coverage', { scanned: last.repos_scanned, expected: last.repos_expected })}
			/>
		</div>

		<Tabs items={tabs} value={tab} {panelId} onChange={(id) => (tab = id)} label={t('maintenance.view')} />

		<div id={panelId} role="tabpanel" aria-label={activeLabel}>
			{#if tab === 'actions'}
				<MaintenanceActions data={actions} />
			{:else if tab === 'drift'}
				<MaintenanceDrift data={drift} />
			{:else if tab === 'graph'}
				<MaintenanceGraph data={graph} />
			{:else if tab === 'conformance'}
				<MaintenanceConformance data={conformance} />
			{:else}
				<MaintenanceProjects {projects} onSaved={refreshActions} />
			{/if}
		</div>
	{/if}
</div>
