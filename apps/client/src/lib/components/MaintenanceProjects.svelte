<script lang="ts">
	import { Alert, Badge, Table } from '@facile/muse';
	import type { MaintenanceProject } from '$lib/backend';
	import { t } from '$lib/i18n/index.svelte';
	import type { AppKey } from '$lib/i18n/messages';
	import MaintenanceTierCell from './MaintenanceTierCell.svelte';

	let { projects, onSaved }: { projects: MaintenanceProject[]; onSaved: () => void } = $props();

	let error = $state('');

	const ciLabels: Record<string, AppKey> = {
		success: 'maintenance.ci.success',
		failure: 'maintenance.ci.failure',
		cancelled: 'maintenance.ci.cancelled',
		timed_out: 'maintenance.ci.timed_out',
		action_required: 'maintenance.ci.action_required',
		startup_failure: 'maintenance.ci.startup_failure',
		skipped: 'maintenance.ci.skipped',
		neutral: 'maintenance.ci.neutral',
		stale: 'maintenance.ci.stale'
	};

	function ciLabel(conclusion: string) {
		if (!conclusion) return t('maintenance.ciUnknown');
		const key = ciLabels[conclusion];
		return key ? t(key) : conclusion;
	}

	function ciTone(conclusion: string) {
		if (conclusion === 'success') return 'success';
		if (['failure', 'timed_out', 'startup_failure'].includes(conclusion)) return 'danger';
		if (conclusion === 'action_required') return 'warning';
		return 'neutral';
	}

	function saved() {
		error = '';
		onSaved();
	}
</script>

<div class="flex flex-col gap-4">
	{#if error}
		<Alert tone="danger" title={error} />
	{/if}
	<Table>
		<thead>
			<tr>
				<th scope="col">{t('maintenance.col.repository')}</th>
				<th scope="col">{t('maintenance.col.tier')}</th>
				<th scope="col">{t('maintenance.col.coverage')}</th>
				<th scope="col">{t('maintenance.col.ci')}</th>
				<th scope="col">{t('maintenance.col.filet')}</th>
				<th scope="col">{t('maintenance.col.open')}</th>
			</tr>
		</thead>
		<tbody>
			{#each projects as project (project.id)}
				<tr>
					<td class="font-medium text-fc-fg">{project.name}</td>
					<td class="min-w-40">
						<MaintenanceTierCell {project} onSaved={saved} onError={(message) => (error = message)} />
					</td>
					<td class="max-w-64">
						{#if project.covered}
							<Badge tone="success">{t('maintenance.covered')}</Badge>
						{:else}
							<div class="flex flex-col gap-1">
								<Badge tone="warning">{t('maintenance.notScanned')}</Badge>
								{#if project.not_covered}
									<span class="text-fc-xs break-words whitespace-normal text-fc-fg-muted">{project.not_covered}</span>
								{/if}
							</div>
						{/if}
					</td>
					<td>
						<Badge tone={ciTone(project.health.ci_conclusion)}>{ciLabel(project.health.ci_conclusion)}</Badge>
					</td>
					<td class="text-fc-sm">
						{#if !project.health.filet_wired}
							<span class="text-fc-fg-muted">{t('maintenance.filetMissing')}</span>
						{:else if project.health.filet_pin}
							<span class="font-mono">{project.health.filet_pin}</span>
						{:else}
							{t('maintenance.filetWired')}
						{/if}
					</td>
					<td class="tabular-nums">{project.open_findings}</td>
				</tr>
			{/each}
		</tbody>
	</Table>
</div>
