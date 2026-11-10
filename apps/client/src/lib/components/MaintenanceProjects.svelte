<script lang="ts">
	import { Alert, Badge, Table } from '@facile/muse';
	import type { MaintenanceProject } from '$lib/backend';
	import { t } from '$lib/i18n/index.svelte';
	import MaintenanceTierCell from './MaintenanceTierCell.svelte';

	let { projects }: { projects: MaintenanceProject[] } = $props();

	let error = $state('');

	function ciTone(conclusion: string) {
		if (conclusion === 'success') return 'success';
		if (conclusion === 'failure' || conclusion === 'timed_out') return 'danger';
		return 'neutral';
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
					<td class="min-w-40"><MaintenanceTierCell {project} onError={(message) => (error = message)} /></td>
					<td>
						{#if project.covered}
							<Badge tone="success">{t('maintenance.covered')}</Badge>
						{:else}
							<Badge tone="warning">{project.not_covered}</Badge>
						{/if}
					</td>
					<td>
						<Badge tone={ciTone(project.health.ci_conclusion)}>
							{project.health.ci_conclusion || t('maintenance.ciUnknown')}
						</Badge>
					</td>
					<td class="font-mono text-fc-sm">
						{project.health.filet_wired ? project.health.filet_pin || '✓' : t('maintenance.filetMissing')}
					</td>
					<td class="tabular-nums">{project.open_findings}</td>
				</tr>
			{/each}
		</tbody>
	</Table>
</div>
