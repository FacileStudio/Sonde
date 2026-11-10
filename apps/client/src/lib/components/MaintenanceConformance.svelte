<script lang="ts">
	import { Badge, EmptyState, Table, icons } from '@facile/muse';
	import type { MaintenanceCheck, MaintenanceConformance } from '$lib/backend';
	import { t } from '$lib/i18n/index.svelte';

	let { data }: { data: MaintenanceConformance } = $props();

	const rows = $derived(
		[...data.projects]
			.sort((a, b) => a.passed / a.total - b.passed / b.total || b.total - b.passed - (a.total - a.passed))
			.map((project) => ({
				...project,
				cells: new Map<string, MaintenanceCheck>(project.checks.map((check) => [check.id, check]))
			}))
	);
</script>

{#if rows.length === 0}
	<EmptyState icon={icons.shield} title={t('maintenance.conformance.empty')} />
{:else}
	<Table>
		<thead>
			<tr>
				<th scope="col">{t('maintenance.conformance.project')}</th>
				<th scope="col">{t('maintenance.conformance.score')}</th>
				{#each data.checks as id (id)}
					<th scope="col" class="font-mono">{id}</th>
				{/each}
			</tr>
		</thead>
		<tbody>
			{#each rows as row (row.id)}
				<tr>
					<th scope="row" class="font-medium text-fc-fg">{row.name}</th>
					<td class="text-fc-sm whitespace-nowrap">{row.passed}/{row.total}</td>
					{#each data.checks as id (id)}
						{@const cell = row.cells.get(id)}
						<td title={cell?.detail ?? ''}>
							{#if !cell}
								<span class="text-fc-xs text-fc-fg-muted">{t('maintenance.conformance.na')}</span>
							{:else}
								<Badge tone={cell.ok ? 'success' : 'danger'}>
									{cell.ok ? t('maintenance.conformance.pass') : t('maintenance.conformance.fail')}
								</Badge>
								<span class="sr-only">{cell.detail}</span>
							{/if}
						</td>
					{/each}
				</tr>
			{/each}
		</tbody>
	</Table>
{/if}
