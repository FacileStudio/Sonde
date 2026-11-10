<script lang="ts">
	import { Badge, EmptyState, Table, Tabs, icons } from '@facile/muse';
	import type { MaintenanceAction, MaintenanceActions, MaintenanceLevel } from '$lib/backend';
	import { MISSING, percent } from '$lib/format';
	import { locale, t } from '$lib/i18n/index.svelte';
	import type { AppKey } from '$lib/i18n/messages';

	let { data }: { data: MaintenanceActions } = $props();

	let level = $state<MaintenanceLevel>('act');

	const panelId = 'maintenance-level-panel';
	const visibleProjects = 3;
	const levelTone = { act: 'danger', schedule: 'warning', track: 'neutral' } as const;
	const levelLabel = $derived({
		act: t('maintenance.level.act'),
		schedule: t('maintenance.level.schedule'),
		track: t('maintenance.level.track')
	});

	const levels = $derived([
		{ id: 'act', label: levelLabel.act, badge: data.counts.act },
		{ id: 'schedule', label: levelLabel.schedule, badge: data.counts.schedule },
		{ id: 'track', label: levelLabel.track, badge: data.counts.track }
	]);
	const shown = $derived(data.actions.filter((action) => action.level === level));
	const names = $derived(new Intl.ListFormat(locale(), { style: 'long', type: 'conjunction' }));

	const severityKeys: Record<string, AppKey> = {
		CRITICAL: 'maintenance.severity.critical',
		HIGH: 'maintenance.severity.high',
		MEDIUM: 'maintenance.severity.medium',
		LOW: 'maintenance.severity.low'
	};

	function severity(raw: string) {
		return t(severityKeys[raw] ?? 'maintenance.severity.unknown');
	}

	function severityTone(raw: string) {
		if (raw === 'CRITICAL') return 'danger';
		if (raw === 'HIGH') return 'warning';
		if (raw === 'MEDIUM') return 'info';
		return 'neutral';
	}

	const rowKey = (action: MaintenanceAction) => `${action.ecosystem}|${action.package}|${action.fixed_in}`;
	const epss = (value: number) => (value > 0 ? percent(value * 100, value < 0.01 ? 2 : 0) : MISSING);
</script>

<div class="flex flex-col gap-6">
	<Tabs items={levels} value={level} {panelId} onChange={(id) => (level = id as MaintenanceLevel)} label={t('maintenance.levels')} />

	<div id={panelId} role="tabpanel" aria-label={levelLabel[level]}>
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
					{#each shown as action (rowKey(action))}
						{@const projectNames = action.projects.map((project) => project.name)}
						<tr>
							<td><Badge tone={levelTone[action.level]}>{levelLabel[action.level]}</Badge></td>
							<td>
								<div class="flex flex-col">
									<span class="font-medium text-fc-fg">{action.package}</span>
									<span class="text-fc-xs text-fc-fg-muted">
										{action.ecosystem} · {action.fixed_in ? `→ ${action.fixed_in}` : t('maintenance.noFix')}
									</span>
								</div>
							</td>
							<td class="text-fc-sm text-fc-fg-muted">
								{names.format(projectNames.slice(0, visibleProjects))}
								{#if projectNames.length > visibleProjects}
									<details class="mt-1">
										<summary class="min-h-11 cursor-pointer text-fc-fg">
											{t('maintenance.more', { count: projectNames.length - visibleProjects })}
										</summary>
										<p class="break-words">{names.format(projectNames.slice(visibleProjects))}</p>
									</details>
								{/if}
							</td>
							<td class="tabular-nums">{action.findings}</td>
							<td><Badge tone={severityTone(action.worst_severity)}>{severity(action.worst_severity)}</Badge></td>
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
</div>
