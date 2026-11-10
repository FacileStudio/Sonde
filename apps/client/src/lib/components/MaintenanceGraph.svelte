<script lang="ts">
	import { EmptyState, StatusDot, Tabs, icons } from '@facile/muse';
	import type { MaintenanceEdge, MaintenanceGraph } from '$lib/backend';
	import {
		GRAPH_LAYERS,
		GRAPH_VIEWS,
		edgePath,
		edgesFor,
		geometry,
		place,
		risky,
		type GraphView,
		type PlacedNode
	} from '$lib/maintenance-graph';
	import { t } from '$lib/i18n/index.svelte';
	import MaintenanceGraphPanel from './MaintenanceGraphPanel.svelte';

	let { data }: { data: MaintenanceGraph } = $props();

	let view = $state<GraphView>('libraries');
	let selected = $state('');

	const canvasId = 'maintenance-graph-canvas';
	const views = $derived(GRAPH_VIEWS.map((id) => ({ id, label: t(`maintenance.graph.view.${id}`) })));
	const layout = $derived(place(data));
	const byName = $derived(new Map(layout.nodes.map((node) => [node.name, node])));
	const shown = $derived(
		edgesFor(data.edges, view).flatMap((edge) => {
			const from = byName.get(edge.from);
			const to = byName.get(edge.to);
			return from && to ? [{ edge, from, to }] : [];
		})
	);
	const degree = $derived(
		shown.reduce((counts, { edge }) => {
			counts.set(edge.from, (counts.get(edge.from) ?? 0) + 1);
			counts.set(edge.to, (counts.get(edge.to) ?? 0) + 1);
			return counts;
		}, new Map<string, number>())
	);
	const node = $derived(selected ? byName.get(selected) : undefined);

	function stroke(edge: MaintenanceEdge) {
		const touched = selected !== '' && (edge.from === selected || edge.to === selected);
		const tone = risky(edge) ? 'stroke-fc-warning' : touched ? 'stroke-fc-accent' : 'stroke-fc-fg-muted';
		const dim = selected !== '' && !touched ? 'opacity-20' : 'opacity-70';
		return `${tone} ${dim}`;
	}

	function uptimeTone(status: PlacedNode['uptime_status']) {
		if (status === 'up') return 'success';
		if (status === 'down') return 'danger';
		return 'neutral';
	}
</script>

{#if data.edges.length === 0}
	<EmptyState icon={icons.layers} title={t('maintenance.graph.empty')} />
{:else}
	<div class="flex flex-col gap-4">
		<div class="flex flex-wrap items-center justify-between gap-3">
			<Tabs
				items={views}
				value={view}
				panelId={canvasId}
				onChange={(id) => (view = id as GraphView)}
				label={t('maintenance.graph.viewLabel')}
			/>
			<p class="text-fc-xs text-fc-fg-muted">{t('maintenance.graph.sharedSecret')}</p>
		</div>
		<div class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_18rem]">
			<div id={canvasId} class="overflow-x-auto rounded-fc-md bg-fc-component p-3">
				<div class="relative" style:width="{layout.width}px">
					<div class="flex">
						{#each GRAPH_LAYERS as layer (layer)}
							<h3 class="text-fc-xs font-semibold text-fc-fg-muted" style:width="{geometry.column}px">
								{t(`maintenance.graph.layer.${layer}`)}
							</h3>
						{/each}
					</div>
					<div class="relative mt-2" style:height="{layout.height}px">
						<svg
							class="pointer-events-none absolute inset-0 fill-none"
							width={layout.width}
							height={layout.height}
							aria-hidden="true"
						>
							{#each shown as { edge, from, to } (`${edge.from}|${edge.to}|${edge.kind}|${edge.evidence}`)}
								<path d={edgePath(from, to)} class={stroke(edge)} stroke-width="1.5" />
							{/each}
						</svg>
						{#each layout.nodes as item (item.name)}
							<button
								type="button"
								class="absolute flex flex-col justify-center gap-0.5 rounded-fc-sm border px-2 text-left transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring {selected ===
								item.name
									? 'border-fc-accent bg-fc-surface'
									: 'border-fc-border bg-fc-bg hover:bg-fc-surface'}"
								style:left="{item.x}px"
								style:top="{item.y}px"
								style:width="{geometry.node}px"
								style:height="{geometry.height}px"
								aria-pressed={selected === item.name}
								aria-label={t('maintenance.graph.nodeLabel', { name: item.name, count: degree.get(item.name) ?? 0 })}
								onclick={() => (selected = selected === item.name ? '' : item.name)}
							>
								<span class="flex items-center gap-1.5 truncate text-fc-sm font-medium text-fc-fg">
									{#if item.uptime_status}
										<StatusDot tone={uptimeTone(item.uptime_status)} />
									{/if}
									<span class="truncate">{item.name}</span>
								</span>
								{#if item.checks_total > 0 || item.open_act > 0}
									<span class="truncate text-fc-xs text-fc-fg-muted">
										{item.checks_total > 0 ? t('maintenance.graph.checks', { ok: item.checks_ok, total: item.checks_total }) : ''}
										{item.open_act > 0 ? `· ${t('maintenance.graph.openAct', { count: item.open_act })}` : ''}
									</span>
								{/if}
							</button>
						{/each}
					</div>
				</div>
			</div>
			<MaintenanceGraphPanel {node} edges={data.edges} />
		</div>
	</div>
{/if}
