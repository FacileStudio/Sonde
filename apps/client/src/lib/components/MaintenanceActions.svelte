<script lang="ts">
	import { Badge, EmptyState, Table, Tabs, icons } from '@facile/muse';
	import type { MaintenanceAction, MaintenanceActions, MaintenanceLevel } from '$lib/backend';
	import { t } from '$lib/i18n/index.svelte';

	let { data }: { data: MaintenanceActions } = $props();

	let level = $state<MaintenanceLevel>('act');

	const levels = $derived([
		{ id: 'act', label: `${t('maintenance.level.act')} · ${data.counts.act}` },
		{ id: 'schedule', label: `${t('maintenance.level.schedule')} · ${data.counts.schedule}` },
		{ id: 'track', label: `${t('maintenance.level.track')} · ${data.counts.track}` }
	]);
	const shown = $derived(data.actions.filter((action) => action.level === level));

	const levelTone = { act: 'danger', schedule: 'warning', track: 'neutral' } as const;

	function severityTone(severity: string) {
		if (severity === 'CRITICAL') return 'danger';
		if (severity === 'HIGH') return 'warning';
		if (severity === 'MEDIUM') return 'info';
		return 'neutral';
	}

	const visibleProjects = 3;

	function projectList(action: MaintenanceAction) {
		const names = action.projects.slice(0, visibleProjects).map((project) => project.name);
		const rest = action.projects.length - visibleProjects;
		return rest > 0 ? `${names.join(', ')} ${t('maintenance.more', { count: rest })}` : names.join(', ');
	}

	const epss = (value: number) => (value > 0 ? `${(value * 100).toFixed(value < 0.01 ? 2 : 0)}%` : '–');
</script>

<div class="flex flex-col gap-6">
	<Tabs items={levels} value={level} onChange={(id) => (level = id as MaintenanceLevel)} label={t('maintenance.levels')} />

	{#if shown.length === 0}
		<EmptyState
			icon={icons.shield}
			title={t('maintenance.actions.empty.title')}
			description={t('maintenance.actions.empty.description')}
		/>
	{:else}
		<Table>
			<thead>
				<tr>
					<th scope="col">{t('maintenance.col.level')}</th>
					<th scope="col">{t('maintenance.col.upgrade')}</th>
					<th scope="col">{t('maintenance.col.projects')}</th>
					<th scope="col">{t('maintenance.col.findings')}</th>
					<th scope="col">{t('maintenance.col.worst')}</th>
					<th scope="col">{t('maintenance.col.exploit')}</th>
				</tr>
			</thead>
			<tbody>
				{#each shown as action (action.ecosystem + action.package + action.fixed_in)}
					<tr>
						<td><Badge tone={levelTone[action.level]}>{t(`maintenance.level.${action.level}`)}</Badge></td>
						<td>
							<div class="flex flex-col">
								<span class="font-medium text-fc-fg">{action.package}</span>
								<span class="text-fc-xs text-fc-fg-muted">
									{action.ecosystem} · {action.fixed_in ? `→ ${action.fixed_in}` : t('maintenance.noFix')}
								</span>
							</div>
						</td>
						<td class="text-fc-sm text-fc-fg-muted">{projectList(action)}</td>
						<td class="tabular-nums">{action.findings}</td>
						<td><Badge tone={severityTone(action.worst_severity)}>{action.worst_severity}</Badge></td>
						<td class="tabular-nums">
							{#if action.kev}
								<Badge tone="danger">{t('maintenance.kev')}</Badge>
							{:else}
								{epss(action.max_epss)}
							{/if}
						</td>
					</tr>
				{/each}
			</tbody>
		</Table>
	{/if}
</div>
