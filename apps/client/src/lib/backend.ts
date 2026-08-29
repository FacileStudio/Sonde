import { browser } from '$app/environment';
import { goto } from '$app/navigation';

const backendBaseUrl = '/api';

export const ssoLoginUrl = `${backendBaseUrl}/auth/oidc`;

export type AuthUser = {
	id: number;
	email: string;
	name: string;
	is_admin: boolean;
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

type ApiErrorPayload = {
	error?: {
		message?: string;
	};
};

/*
 * 401 is the only code that means "sign in again". porte answers 401 for an absent, invalid
 * or expired session and reserves 403 for two states that are not about being signed out: a
 * cookie mutation missing X-Facile-CSRF, and this app's admin gate on the status-page and
 * export routes. Bouncing a signed-in non-admin to /login would land them on a page that
 * immediately sends them back, so 403 stays with the caller.
 */
function sessionRefused() {
	if (!browser) return;
	const here = window.location.pathname;
	/* The login page itself probes /auth/me, and a 401 there is the expected answer. */
	if (here === '/login') return;
	const next = here + window.location.search;
	void goto(`/login?next=${encodeURIComponent(next)}`, { replaceState: true });
}

async function apiFetch<T>(path: string, options: RequestInit = {}): Promise<T> {
	const headers = new Headers(options.headers);
	if (!headers.has('Content-Type') && options.body) {
		headers.set('Content-Type', 'application/json');
	}
	const mutating = options.method && options.method !== 'GET';
	if (mutating) {
		headers.set('X-Facile-CSRF', '1');
	}

	const response = await fetch(`${backendBaseUrl}${path}`, {
		...options,
		headers,
		credentials: 'same-origin'
	});
	if (!response.ok) {
		if (response.status === 401) sessionRefused();
		let payload: ApiErrorPayload | undefined;
		try {
			payload = (await response.json()) as ApiErrorPayload;
		} catch {
			payload = undefined;
		}
		const message = payload?.error?.message || `Request failed with status ${response.status}`;
		throw Object.assign(new Error(message), { status: response.status });
	}
	if (response.status === 204) {
		return undefined as T;
	}
	return (await response.json()) as T;
}

function download(path: string, filename: string) {
	return fetch(`${backendBaseUrl}${path}`, { credentials: 'same-origin' }).then(async (response) => {
		if (!response.ok) throw new Error(`Request failed with status ${response.status}`);
		const blob = await response.blob();
		const url = URL.createObjectURL(blob);
		const anchor = document.createElement('a');
		anchor.href = url;
		anchor.download = filename;
		anchor.click();
		URL.revokeObjectURL(url);
	});
}

async function toError(response: Response, fallback: string): Promise<Error> {
	let payload: ApiErrorPayload | undefined;
	try {
		payload = (await response.json()) as ApiErrorPayload;
	} catch {
		payload = undefined;
	}
	return new Error(payload?.error?.message || fallback);
}

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

export const backend = {
	baseUrl: backendBaseUrl,
	authConfig() {
		return apiFetch<AuthConfig>('/auth/config');
	},
	me() {
		return apiFetch<MeResponse>('/auth/me');
	},
	logout() {
		return apiFetch<void>('/auth/logout', { method: 'POST' });
	},
	updateProfile(input: ProfileUpdate) {
		return apiFetch<{ user: Omit<AuthUser, 'created_at'> }>('/users/me', {
			method: 'PATCH',
			body: JSON.stringify(input)
		});
	},
	listMonitors() {
		return apiFetch<Monitor[]>('/monitors');
	},
	createMonitor(input: MonitorInput) {
		return apiFetch<Monitor>('/monitors', { method: 'POST', body: JSON.stringify(input) });
	},
	getMonitor(id: number) {
		return apiFetch<Monitor>(`/monitors/${id}`);
	},
	updateMonitor(id: number, input: MonitorInput) {
		return apiFetch<Monitor>(`/monitors/${id}`, { method: 'PATCH', body: JSON.stringify(input) });
	},
	deleteMonitor(id: number) {
		return apiFetch<void>(`/monitors/${id}`, { method: 'DELETE' });
	},
	monitorUptime(id: number, window: UptimeWindow) {
		return apiFetch<UptimeResponse>(`/monitors/${id}/uptime?window=${window}`);
	},
	/* One request for all four windows. The service computed them together
	   anyway; asking four times was the client's idea, not the server's. */
	monitorUptimeSummary(id: number) {
		return apiFetch<UptimeResponse[]>(`/monitors/${id}/uptime/summary`);
	},
	/* The whole monitors page in one request. Fetching per-monitor checks from
	   the browser was an N+1 that got worse with every monitor added. */
	monitorsOverview() {
		return apiFetch<Fleet>('/monitors/overview');
	},
	monitorHistory(id: number, window: UptimeWindow) {
		return apiFetch<MonitorHistory>(`/monitors/${id}/history?window=${window}`);
	},
	listIncidents() {
		return apiFetch<Incident[]>('/incidents');
	},
	incidentStats(days = 30) {
		return apiFetch<IncidentStats>(`/incidents/stats?days=${days}`);
	},
	integrations() {
		return apiFetch<Integrations>('/integrations');
	},
	monitorIncidents(id: number) {
		return apiFetch<Incident[]>(`/monitors/${id}/incidents`);
	},
	monitorChecks(id: number, limit = 50) {
		return apiFetch<Check[]>(`/monitors/${id}/checks?limit=${limit}`);
	},
	exportConfig() {
		return download('/config/export', 'sonde-config.json');
	},
	listStatusPages() {
		return apiFetch<StatusPage[]>('/status-pages');
	},
	createStatusPage(input: StatusPageInput) {
		return apiFetch<StatusPage>('/status-pages', { method: 'POST', body: JSON.stringify(input) });
	},
	getStatusPage(id: number) {
		return apiFetch<StatusPage>(`/status-pages/${id}`);
	},
	updateStatusPage(id: number, input: StatusPageInput) {
		return apiFetch<StatusPage>(`/status-pages/${id}`, {
			method: 'PATCH',
			body: JSON.stringify(input)
		});
	},
	deleteStatusPage(id: number) {
		return apiFetch<void>(`/status-pages/${id}`, { method: 'DELETE' });
	},
	listWebhooks() {
		return apiFetch<Webhook[]>('/webhooks');
	},
	createWebhook(input: WebhookInput) {
		return apiFetch<Webhook>('/webhooks', { method: 'POST', body: JSON.stringify(input) });
	},
	getWebhook(id: number) {
		return apiFetch<Webhook>(`/webhooks/${id}`);
	},
	updateWebhook(id: number, input: WebhookInput) {
		return apiFetch<Webhook>(`/webhooks/${id}`, { method: 'PATCH', body: JSON.stringify(input) });
	},
	deleteWebhook(id: number) {
		return apiFetch<void>(`/webhooks/${id}`, { method: 'DELETE' });
	},
	testWebhook(idOrInput: number | WebhookInput) {
		if (typeof idOrInput === 'number') {
			return apiFetch<WebhookTestResult>(`/webhooks/${idOrInput}/test`, { method: 'POST' });
		}
		return apiFetch<WebhookTestResult>('/webhooks/test', {
			method: 'POST',
			body: JSON.stringify(idOrInput)
		});
	},
	publicStatusPage(slug: string) {
		return fetch(`${backendBaseUrl}/public/status/${slug}`).then(async (response) => {
			if (!response.ok) throw await toError(response, 'Status page not found');
			return (await response.json()) as PublicStatusPage;
		});
	}
};
