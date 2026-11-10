<script lang="ts">
	import { Badge, Card } from '@facile/muse';
	import type { MaintenanceEdge, MaintenanceNode } from '$lib/backend';
	import { risky } from '$lib/maintenance-graph';
	import { t } from '$lib/i18n/index.svelte';

	let { node, edges }: { node: MaintenanceNode | undefined; edges: MaintenanceEdge[] } = $props();

	const dependsOn = $derived(node ? edges.filter((edge) => edge.from === node.name) : []);
	const usedBy = $derived(node ? edges.filter((edge) => edge.to === node.name) : []);
	const facts = $derived(
		node
			? [
					node.deployed_version && t('maintenance.graph.deployed', { version: node.deployed_version }),
					node.latest_tag && t('maintenance.graph.latest', { version: node.latest_tag }),
					node.checks_total > 0 && t('maintenance.graph.checks', { ok: node.checks_ok, total: node.checks_total })
				]
					.filter(Boolean)
					.join(' · ')
			: ''
	);
	const sections = $derived([
		{ title: t('maintenance.graph.dependsOn'), list: dependsOn, other: (edge: MaintenanceEdge) => edge.to },
		{ title: t('maintenance.graph.usedBy'), list: usedBy, other: (edge: MaintenanceEdge) => edge.from }
	]);
</script>

<Card class="flex flex-col gap-4 self-start">
	{#if !node}
		<p class="text-fc-sm text-fc-fg-muted">{t('maintenance.graph.pick')}</p>
	{:else}
		<div class="flex flex-col gap-1.5">
			<h3 class="text-fc-md font-semibold break-words text-fc-fg">{node.name}</h3>
			<div class="flex flex-wrap gap-1.5">
				{#if node.id === 0}
					<Badge tone="neutral">{t('maintenance.graph.notListed')}</Badge>
				{:else if node.tier}
					<Badge tone="neutral">{t(`maintenance.tier.${node.tier}`)}</Badge>
				{/if}
				{#if node.uptime_status}
					<Badge tone={node.uptime_status === 'up' ? 'success' : node.uptime_status === 'down' ? 'danger' : 'neutral'}>
						{t(`maintenance.graph.uptime.${node.uptime_status}`)}
					</Badge>
				{/if}
				{#if node.open_act > 0}
					<Badge tone="danger">{t('maintenance.graph.openAct', { count: node.open_act })}</Badge>
				{/if}
			</div>
			{#if facts}
				<p class="text-fc-xs text-fc-fg-muted">{facts}</p>
			{/if}
		</div>
		{#each sections as section (section.title)}
			<div class="flex flex-col gap-2">
				<h4 class="text-fc-xs font-semibold text-fc-fg-muted">{section.title}</h4>
				{#if section.list.length === 0}
					<p class="text-fc-sm text-fc-fg-muted">{t('maintenance.graph.none')}</p>
				{:else}
					<ul class="flex flex-col gap-1.5">
						{#each section.list as edge (`${edge.from}|${edge.to}|${edge.kind}|${edge.evidence}`)}
							<li class="flex flex-wrap items-baseline gap-x-2 gap-y-0.5 text-fc-sm">
								<span class="font-medium text-fc-fg">{section.other(edge)}</span>
								<Badge tone={risky(edge) ? 'warning' : 'neutral'}>{t(`maintenance.graph.kind.${edge.kind}`)}</Badge>
								<span class="font-mono text-fc-xs break-all text-fc-fg-muted">{edge.evidence}</span>
							</li>
						{/each}
					</ul>
				{/if}
			</div>
		{/each}
	{/if}
</Card>
