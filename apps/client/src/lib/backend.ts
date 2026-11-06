import { browser } from '$app/environment';
import { goto } from '$app/navigation';
import type {
	Bucket,
	Check,
	Fleet,
	Incident,
	IncidentStats,
	Integrations,
	MeResponse,
	Monitor,
	MonitorHistory,
	MonitorInput,
	ProfileUpdate,
	PublicStatusPage,
	StatusPage,
	StatusPageInput,
	UptimeResponse,
	UptimeWindow,
	AuthUser,
	Webhook,
	WebhookInput,
	WebhookTestResult
} from '$lib/types';

/* Re-exported so every call site keeps importing its types from the client it
   calls. Splitting the file was a size fix, not an API change. */
export type * from '$lib/types';
export { UPTIME_WINDOWS } from '$lib/types';

const backendBaseUrl = '/api';

export const ssoLoginUrl = `${backendBaseUrl}/auth/oidc`;

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

async function apiFetch<T>(
	path: string,
	options: RequestInit = {},
	silent = false
): Promise<T> {
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
		if (response.status === 401 && !silent) sessionRefused();
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
	me() {
		return apiFetch<MeResponse>('/auth/me');
	},
	probeSession() {
		return apiFetch<MeResponse>('/auth/me', {}, true);
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
