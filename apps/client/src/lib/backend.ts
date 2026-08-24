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

export type Incident = {
	id: number;
	monitor_id: number;
	opened_at: string;
	resolved_at: string | null;
	cause: string;
};

export type UptimeResponse = {
	window: string;
	uptime: number | null;
	total: number;
	failed: number;
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
	monitorUptime(id: number, window: '24h' | '7d') {
		return apiFetch<UptimeResponse>(`/monitors/${id}/uptime?window=${window}`);
	},
	listIncidents() {
		return apiFetch<Incident[]>('/incidents');
	},
	monitorIncidents(id: number) {
		return apiFetch<Incident[]>(`/monitors/${id}/incidents`);
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
	publicStatusPage(slug: string) {
		return fetch(`${backendBaseUrl}/public/status/${slug}`).then(async (response) => {
			if (!response.ok) throw await toError(response, 'Status page not found');
			return (await response.json()) as PublicStatusPage;
		});
	}
};
