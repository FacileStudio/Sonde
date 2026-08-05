const backendBaseUrl = (import.meta.env.VITE_API_BASE_URL as string | undefined)?.replace(/\/$/, '') ||
	'http://localhost:4000';

export type AuthResponse = {
	user_id: string;
	token: string;
};

export type MeResponse = {
	user: {
		id: string;
		email: string;
	};
};

export type EventRecord = {
	id: number;
	name: string;
	starts: string;
	ends?: string;
	owner_id: string;
};

export type ListEventsResponse = {
	events: EventRecord[];
};

export type GenerateTicketResponse = {
	code: string;
	ticket: {
		id: number;
		event_id: number;
		status: string;
		used_at?: string;
	};
};

export type ValidateTicketResponse = {
	valid: boolean;
	status: string;
	ticket: {
		id: number;
		event_id: number;
		status: string;
		used_at?: string;
	};
};

type ApiErrorPayload = {
	error?: {
		message?: string;
	};
};

async function apiFetch<T>(path: string, options: RequestInit = {}, token?: string) {
	const headers = new Headers(options.headers);
	if (!headers.has('Content-Type') && options.body) {
		headers.set('Content-Type', 'application/json');
	}
	if (token) {
		headers.set('Authorization', `Bearer ${token}`);
	}

	const response = await fetch(`${backendBaseUrl}${path}`, { ...options, headers });
	if (!response.ok) {
		let payload: ApiErrorPayload | undefined;
		try {
			payload = (await response.json()) as ApiErrorPayload;
		} catch {
			payload = undefined;
		}
		throw new Error(payload?.error?.message || `Request failed with status ${response.status}`);
	}
	return (await response.json()) as T;
}

export const backend = {
	baseUrl: backendBaseUrl,
	register(email: string, password: string) {
		return apiFetch<AuthResponse>('/auth/register', {
			method: 'POST',
			body: JSON.stringify({ email, password })
		});
	},
	login(email: string, password: string) {
		return apiFetch<AuthResponse>('/auth/login', {
			method: 'POST',
			body: JSON.stringify({ email, password })
		});
	},
	me(token: string) {
		return apiFetch<MeResponse>('/users/me', {}, token);
	},
	listEvents() {
		return apiFetch<ListEventsResponse>('/events');
	},
	createEvent(token: string, payload: { name: string; starts: string; ends?: string }) {
		return apiFetch<{ id: number }>(
			'/events',
			{ method: 'POST', body: JSON.stringify(payload) },
			token
		);
	},
	generateTicket(token: string, eventId: number) {
		return apiFetch<GenerateTicketResponse>(`/events/${eventId}/tickets`, { method: 'POST' }, token);
	},
	validateTicket(code: string) {
		return apiFetch<ValidateTicketResponse>('/tickets/validate', {
			method: 'POST',
			body: JSON.stringify({ code })
		});
	},
	checkInTicket(token: string, code: string) {
		return apiFetch<{ ticket: ValidateTicketResponse['ticket'] }>(
			'/tickets/checkin',
			{ method: 'POST', body: JSON.stringify({ code }) },
			token
		);
	}
};
