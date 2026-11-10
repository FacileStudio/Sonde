import type { MaintenanceEdge, MaintenanceEdgeKind, MaintenanceGraph, MaintenanceNode } from '$lib/maintenance-types';

/** The columns of the fleet graph, left to right. */
export type GraphLayer = 'clients' | 'apps' | 'platform' | 'identity' | 'libraries';

/** Which edges the graph draws at once. Events also carry identity and log shipping. */
export type GraphView = 'libraries' | 'calls' | 'events';

export const GRAPH_LAYERS: GraphLayer[] = ['clients', 'apps', 'platform', 'identity', 'libraries'];

export const GRAPH_VIEWS: GraphView[] = ['libraries', 'calls', 'events'];

const viewKinds: Record<GraphView, MaintenanceEdgeKind[]> = {
	libraries: ['lib'],
	calls: ['call'],
	events: ['event', 'identity', 'logs']
};

const platform = ['antenne', 'journal', 'sonde', 'bulle', 'sonar'];
const libraries = ['tronc', 'porte', 'muse', 'antenne-client'];

export const geometry = { column: 200, row: 56, node: 152, height: 44, pad: 8 };

/** Places a node in its column: named platform pieces and libraries first, then its tier. */
export function layerOf(node: MaintenanceNode): GraphLayer {
	const name = node.name.toLowerCase();
	if (platform.includes(name)) return 'platform';
	if (name === 'registre') return 'identity';
	if (libraries.includes(name) || node.tier === 'library') return 'libraries';
	if (node.tier === 'client') return 'clients';
	return 'apps';
}

/** A positioned node, its top-left corner in SVG units. */
export type PlacedNode = MaintenanceNode & { x: number; y: number };

/** Lays the nodes out in columns, sorted by name inside each. */
export function place(graph: MaintenanceGraph): { nodes: PlacedNode[]; width: number; height: number } {
	const nodes: PlacedNode[] = [];
	let rows = 1;
	GRAPH_LAYERS.forEach((layer, column) => {
		const members = graph.nodes
			.filter((node) => layerOf(node) === layer)
			.sort((a, b) => a.name.localeCompare(b.name));
		rows = Math.max(rows, members.length);
		members.forEach((node, row) => {
			nodes.push({ ...node, x: column * geometry.column + geometry.pad, y: row * geometry.row + geometry.pad });
		});
	});
	return { nodes, width: GRAPH_LAYERS.length * geometry.column, height: rows * geometry.row + geometry.pad };
}

/** The edges one view draws. */
export function edgesFor(edges: MaintenanceEdge[], view: GraphView): MaintenanceEdge[] {
	return edges.filter((edge) => viewKinds[view].includes(edge.kind));
}

/** A call authenticated by a shared secret is drawn in the warning tone. */
export function risky(edge: MaintenanceEdge): boolean {
	return edge.kind === 'call' && (edge.auth === 'static_key' || edge.auth === 'webhook');
}

/** A cubic path from the side of one node facing the other, looping out when both share a column. */
export function edgePath(from: PlacedNode, to: PlacedNode): string {
	const fy = from.y + geometry.height / 2;
	const ty = to.y + geometry.height / 2;
	if (from.x === to.x) {
		const x = from.x + geometry.node;
		return `M ${x} ${fy} C ${x + 40} ${fy}, ${x + 40} ${ty}, ${x} ${ty}`;
	}
	const forward = to.x > from.x;
	const fx = forward ? from.x + geometry.node : from.x;
	const tx = forward ? to.x : to.x + geometry.node;
	const bend = (tx - fx) / 2;
	return `M ${fx} ${fy} C ${fx + bend} ${fy}, ${tx - bend} ${ty}, ${tx} ${ty}`;
}
