/*
 * The wire types. One file, because they are the contract with apps/api and
 * reading them next to each other is how a drift gets spotted.
 *
 * Import them from $lib/backend, not from here: the client re-exports the lot,
 * so a type always arrives alongside the function that returns it.
 */

export type AuthUser = {
	id: number;
	email: string;
	name: string;
	is_admin: boolean;
	avatar_url: string;
	created_at: string;
};

export type MeResponse = {
	user: AuthUser;
};

/* PATCH /users/me takes either field on its own, and rejects a request carrying neither. */
export type ProfileUpdate = {
	name?: string;
	email?: string;
};

export type AuthConfig = {
	sso_only: boolean;
	oidc_enabled: boolean;
	allow_registration: boolean;
};

export type Monitor = {
	id: number;
	slug: string;
	name: string;
	type: 'http' | 'tcp' | 'push';
	target: string;
	interval_seconds: number;
	timeout_seconds: number;
	expected_status: number;
	expected_keyword: string;
	enabled: boolean;
	push_token?: string | null;
	created_at: string;
	updated_at: string;
};

export type MonitorInput = {
	slug: string;
	name: string;
	type: Monitor['type'];
	target: string;
	interval_seconds: number;
	timeout_seconds: number;
	expected_status: number;
	expected_keyword: string;
	enabled: boolean;
};

export type Check = {
	id: number;
	monitor_id: number;
	ok: boolean;
	latency_ms: number;
	error?: string;
	created_at: string;
};

export type Incident = {
	id: number;
	monitor_id: number;
	opened_at: string;
	resolved_at: string | null;
	cause: string;
};

export type UptimeWindow = '24h' | '7d' | '30d' | '90d';

export const UPTIME_WINDOWS: UptimeWindow[] = ['24h', '7d', '30d', '90d'];

export type UptimeResponse = {
	window: string;
	uptime: number | null;
	total: number;
	failed: number;
};

/* up, down and unknown come from the last probe; paused wins over all three,
   because "nobody is watching this" is not the same fact as "this failed". */
export type MonitorStatus = 'up' | 'down' | 'paused' | 'unknown';

/* A bucket with no checks reports uptime null, not 0. The difference is a gap
   in coverage versus an outage, and they are not the same colour. */
export type Bucket = {
	bucket_start: string;
	total: number;
	failed: number;
	avg_latency_ms: number;
	uptime: number | null;
};

export type MonitorSummary = {
	id: number;
	slug: string;
	name: string;
	type: Monitor['type'];
	target: string;
	interval_seconds: number;
	enabled: boolean;
	status: MonitorStatus;
	last_check_at: string | null;
	last_latency_ms: number | null;
	last_error?: string;
	uptime_24h: number | null;
	avg_latency_24h: number | null;
	checks_24h: number;
	failed_24h: number;
	open_incidents: number;
	history: Bucket[];
};

export type FleetTotals = {
	monitors: number;
	up: number;
	down: number;
	paused: number;
	unknown: number;
	open_incidents: number;
	uptime_24h: number | null;
	avg_latency_24h: number | null;
	checks_24h: number;
};

export type Fleet = {
	totals: FleetTotals;
	monitors: MonitorSummary[];
};

export type MonitorHistory = {
	monitor_id: number;
	window: UptimeWindow;
	resolution: 'hour' | 'day';
	buckets: Bucket[];
};

export type IncidentDay = {
	day: string;
	opened: number;
	downtime_seconds: number;
};

export type IncidentMonitorLoad = {
	monitor_id: number;
	slug: string;
	name: string;
	incidents: number;
	downtime_seconds: number;
};

export type IncidentStats = {
	days: number;
	total: number;
	open: number;
	resolved: number;
	mttr_seconds: number | null;
	longest_seconds: number | null;
	total_downtime_seconds: number;
	by_day: IncidentDay[];
	by_monitor: IncidentMonitorLoad[];
};

/* Not a boolean. "Not connected" hides three situations with three different
   fixes, so settings renders four states and only the in-flight ones pulse. */
export type BusState = 'disabled' | 'connecting' | 'connected' | 'reconnecting';

export type Bus = {
	configured: boolean;
	state: BusState;
	instance: string;
	app: string;
	instance_id?: string;
	channels: string[];
	published: number;
	dropped: number;
	last_event_at: string | null;
	last_error?: string;
};

export type Integrations = {
	bus: Bus;
	journal: { configured: boolean; url?: string };
	retention: { raw_days: number; hourly_months: number; failure_threshold: number };
};

export type StatusPageMonitorLine = {
	id: number;
	slug: string;
	name: string;
	position: number;
};

export type StatusPage = {
	id: number;
	slug: string;
	title: string;
	monitors: StatusPageMonitorLine[];
	created_at: string;
};

export type StatusPageInput = {
	slug: string;
	title: string;
	monitor_ids: number[];
};

export type PublicIncident = {
	id: number;
	opened_at: string;
	cause: string;
};

export type PublicMonitor = {
	id: number;
	slug: string;
	name: string;
	current_status: 'up' | 'down' | 'unknown';
	uptime_24h: number | null;
	uptime_7d: number | null;
	open_incidents: PublicIncident[];
};

export type PublicStatusPage = {
	slug: string;
	title: string;
	monitors: PublicMonitor[];
};

export type Webhook = {
	id: number;
	owner_id: number;
	name: string;
	url: string;
	secret?: string;
	secret_header: string;
	enabled: boolean;
	created_at: string;
	updated_at: string;
};

export type WebhookInput = {
	name?: string;
	url: string;
	secret?: string;
	secret_header?: string;
	enabled?: boolean;
};

export type WebhookTestResult = {
	ok: boolean;
	status_code: number;
	latency_ms: number;
	error?: string;
};
