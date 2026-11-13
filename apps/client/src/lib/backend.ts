import { createApi, redirectToLogin } from '@facile/client';
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
import type {
	MaintenanceActions,
	MaintenanceConformance,
	MaintenanceDrift,
	MaintenanceGraph,
	MaintenanceProject,
	MaintenanceRun,
	MaintenanceTier,
	ProjectRef
} from '$lib/maintenance-types';

/* Re-exported so every call site keeps importing its types from the client it
   calls. Splitting the file was a size fix, not an API change. */
export type * from '$lib/types';
export type * from '$lib/maintenance-types';
export { UPTIME_WINDOWS } from '$lib/types';
export { MAINTENANCE_TIERS } from '$lib/maintenance-types';

const backendBaseUrl = '/api';

export const ssoLoginUrl = `${backendBaseUrl}/auth/oidc`;

const apiFetch = createApi({ base: backendBaseUrl, onUnauthorized: redirectToLogin });

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

export const backend = {
	baseUrl: backendBaseUrl,
	me() {
		return apiFetch<MeResponse>('/auth/me');
	},
	probeSession() {
		return apiFetch<MeResponse>('/auth/me', {}, { quiet: true });
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
	linkMonitorProject(id: number, projectId: number | null) {
		return apiFetch<Monitor>(`/monitors/${id}/project`, {
			method: 'PUT',
			body: JSON.stringify({ project_id: projectId })
		});
	},
	listProjects() {
		return apiFetch<ProjectRef[]>('/projects');
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
	maintenanceRuns(silent = false) {
		return apiFetch<MaintenanceRun[]>('/fleet/runs', {}, { quiet: silent });
	},
	rescanFleet() {
		return apiFetch<{ queued: boolean; run_url?: string }>('/fleet/scans', { method: 'POST' });
	},
	maintenanceActions() {
		return apiFetch<MaintenanceActions>('/fleet/actions');
	},
	maintenanceDrift() {
		return apiFetch<MaintenanceDrift>('/fleet/drift');
	},
	maintenanceGraph() {
		return apiFetch<MaintenanceGraph>('/fleet/graph');
	},
	maintenanceConformance() {
		return apiFetch<MaintenanceConformance>('/fleet/conformance');
	},
	maintenanceProjects() {
		return apiFetch<MaintenanceProject[]>('/fleet/projects');
	},
	setProjectTier(id: number, tier: MaintenanceTier) {
		return apiFetch<{ id: number; tier: MaintenanceTier }>(`/fleet/projects/${id}`, {
			method: 'PATCH',
			body: JSON.stringify({ tier })
		});
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
		return apiFetch<PublicStatusPage>(`/public/status/${slug}`, {}, { quiet: true });
	}
};
