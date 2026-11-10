/** Shapes the fleet maintenance API returns, read by the maintenance page. */
export type MaintenanceLevel = 'act' | 'schedule' | 'track';

export type MaintenanceTier = 'unclassified' | 'deployed' | 'library' | 'cli' | 'client' | 'experiment' | 'archive';

export const MAINTENANCE_TIERS: MaintenanceTier[] = ['unclassified', 'deployed', 'library', 'cli', 'client', 'experiment', 'archive'];

export type MaintenanceAction = {
	level: MaintenanceLevel;
	ecosystem: string;
	package: string;
	fixed_in: string;
	findings: number;
	worst_severity: string;
	kev: boolean;
	max_epss: number;
	first_seen: string;
	advisories: string[];
	projects: { id: number; name: string; tier: MaintenanceTier; installed: string }[];
};

export type MaintenanceActions = {
	actions: MaintenanceAction[];
	counts: Record<MaintenanceLevel, number>;
};

export type MaintenanceDriftProject = {
	id: number;
	name: string;
	held: boolean;
	reason?: string;
	expires_at?: string;
};

export type MaintenanceDrift = {
	packages: {
		package: string;
		latest: string;
		current: number;
		behind: { resolved: string; projects: MaintenanceDriftProject[] }[] | null;
	}[];
};

export type MaintenanceProject = {
	id: number;
	github_repo_id: number;
	name: string;
	default_branch: string;
	tier: MaintenanceTier;
	version_url: string;
	journal_app: string;
	covered: boolean;
	not_covered: string;
	last_scanned_at: string | null;
	open_findings: number;
	health: {
		ci_conclusion: string;
		filet_wired: boolean;
		filet_pin: string;
		deployed_version: string;
		go_directive: string;
	};
};

export type MaintenanceRun = {
	id: number;
	started_at: string;
	finished_at: string;
	douane_version: string;
	repos_expected: number;
	repos_scanned: number;
	complete: boolean;
	findings_open: number;
	received_at: string;
};

export type MaintenanceEdgeKind = 'lib' | 'call' | 'event' | 'identity' | 'logs';

export type MaintenanceEdge = {
	from: string;
	to: string;
	kind: MaintenanceEdgeKind;
	auth: '' | 'audience' | 'static_key' | 'webhook' | 'url';
	evidence: string;
};

export type MaintenanceNode = {
	id: number;
	name: string;
	tier: MaintenanceTier | '';
	deployed_version: string;
	latest_tag: string;
	uptime_status: '' | 'up' | 'down' | 'unknown';
	checks_ok: number;
	checks_total: number;
	open_act: number;
};

export type MaintenanceGraph = { nodes: MaintenanceNode[]; edges: MaintenanceEdge[] };

export type MaintenanceCheck = { id: string; ok: boolean; detail: string };

export type MaintenanceConformance = {
	checks: string[];
	projects: {
		id: number;
		name: string;
		tier: MaintenanceTier;
		passed: number;
		total: number;
		checks: MaintenanceCheck[];
	}[];
};
