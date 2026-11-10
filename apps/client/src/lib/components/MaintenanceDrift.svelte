<script lang="ts">
	import { Badge, Card, EmptyState, icons } from '@facile/muse';
	import type { MaintenanceDrift } from '$lib/backend';
	import { date } from '$lib/format';
	import { t } from '$lib/i18n/index.svelte';

	let { data }: { data: MaintenanceDrift } = $props();

	const packages = $derived(data.packages.filter((row) => row.current > 0 || (row.behind?.length ?? 0) > 0));
	const shortName = (name: string) => name.replace(/^github\.com\/FacileStudio\//, '');
</script>

{#if packages.length === 0}
	<EmptyState icon={icons.layers} title={t('maintenance.drift.empty')} />
{:else}
	<div class="grid gap-4 lg:grid-cols-2">
		{#each packages as row (row.package)}
			<Card class="flex flex-col gap-4">
				<div class="flex flex-wrap items-baseline justify-between gap-2">
					<h3 class="text-fc-md font-semibold text-fc-fg">{shortName(row.package)}</h3>
					<span class="text-fc-xs text-fc-fg-muted">
						{t('maintenance.drift.latest', { version: row.latest })} · {t('maintenance.drift.current', { count: row.current })}
					</span>
				</div>
				{#if !row.behind || row.behind.length === 0}
					<p class="text-fc-sm text-fc-fg-muted">{t('maintenance.drift.allCurrent')}</p>
				{:else}
					<ul class="flex flex-col gap-3">
						{#each row.behind as version (version.resolved)}
							<li class="flex flex-col gap-1">
								<span class="font-mono text-fc-sm text-fc-fg">{version.resolved}</span>
								<div class="flex flex-wrap gap-1.5">
									{#each version.projects as project (project.id)}
										<Badge
											tone={project.held ? 'info' : 'warning'}
											title={project.held && project.expires_at
												? t('maintenance.drift.heldUntil', { date: date(project.expires_at), reason: project.reason ?? '' })
												: undefined}
										>
											{project.name}{project.held ? ` · ${t('maintenance.drift.held')}` : ''}
										</Badge>
									{/each}
								</div>
							</li>
						{/each}
					</ul>
				{/if}
			</Card>
		{/each}
	</div>
{/if}
