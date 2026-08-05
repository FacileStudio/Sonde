<script lang="ts">
	import { onMount } from 'svelte';
	import { CalendarClock, KeyRound, ScanQrCode, Ticket, UserRound } from 'lucide-svelte';
	import { backend, type EventRecord, type ValidateTicketResponse } from '$lib/backend';
	import Button from '$lib/components/ui/button.svelte';
	import Card from '$lib/components/ui/card.svelte';
	import Input from '$lib/components/ui/input.svelte';
	import Label from '$lib/components/ui/label.svelte';

	const tokenStorageKey = 'chime.frontend.token';

	let token = $state('');
	let email = $state('');
	let password = $state('');
	let currentUser = $state<{ id: string; email: string } | null>(null);
	let authMessage = $state('');
	let busyAuth = $state(false);

	let events = $state<EventRecord[]>([]);
	let selectedEventId = $state('');
	let eventName = $state('');
	let eventStarts = $state('');
	let eventEnds = $state('');
	let eventsMessage = $state('');
	let busyEvents = $state(false);

	let generatedCode = $state('');
	let ticketMessage = $state('');
	let validationCode = $state('');
	let validationResult = $state<ValidateTicketResponse | null>(null);
	let validationMessage = $state('');
	let checkInCode = $state('');
	let checkInMessage = $state('');

	onMount(async () => {
		const storedToken = localStorage.getItem(tokenStorageKey) || '';
		token = storedToken;
		await loadEvents();
		if (storedToken) {
			await loadCurrentUser();
		}
	});

	async function loadCurrentUser() {
		if (!token) {
			currentUser = null;
			return;
		}
		try {
			const response = await backend.me(token);
			currentUser = response.user;
		} catch (error) {
			persistToken('');
			currentUser = null;
			authMessage = error instanceof Error ? error.message : 'Failed to load user';
		}
	}

	async function loadEvents() {
		busyEvents = true;
		try {
			const response = await backend.listEvents();
			events = response.events;
			if (!selectedEventId && response.events[0]) {
				selectedEventId = String(response.events[0].id);
			}
		} catch (error) {
			eventsMessage = error instanceof Error ? error.message : 'Failed to load events';
		} finally {
			busyEvents = false;
		}
	}

	function persistToken(nextToken: string) {
		token = nextToken;
		if (nextToken) {
			localStorage.setItem(tokenStorageKey, nextToken);
		} else {
			localStorage.removeItem(tokenStorageKey);
		}
	}

	async function register() {
		busyAuth = true;
		authMessage = '';
		try {
			const response = await backend.register(email, password);
			persistToken(response.token);
			await loadCurrentUser();
			authMessage = 'Account created.';
		} catch (error) {
			authMessage = error instanceof Error ? error.message : 'Registration failed';
		} finally {
			busyAuth = false;
		}
	}

	async function login() {
		busyAuth = true;
		authMessage = '';
		try {
			const response = await backend.login(email, password);
			persistToken(response.token);
			await loadCurrentUser();
			authMessage = 'Logged in.';
		} catch (error) {
			authMessage = error instanceof Error ? error.message : 'Login failed';
		} finally {
			busyAuth = false;
		}
	}

	function logout() {
		persistToken('');
		currentUser = null;
		authMessage = 'Logged out.';
	}

	async function createEvent() {
		if (!token) {
			eventsMessage = 'Log in first.';
			return;
		}
		busyEvents = true;
		eventsMessage = '';
		try {
			await backend.createEvent(token, {
				name: eventName,
				starts: new Date(eventStarts).toISOString(),
				ends: eventEnds ? new Date(eventEnds).toISOString() : undefined
			});
			eventName = '';
			eventStarts = '';
			eventEnds = '';
			eventsMessage = 'Event created.';
			await loadEvents();
		} catch (error) {
			eventsMessage = error instanceof Error ? error.message : 'Failed to create event';
		} finally {
			busyEvents = false;
		}
	}

	async function generateTicket() {
		if (!token || !selectedEventId) {
			ticketMessage = 'Pick an event and log in first.';
			return;
		}
		try {
			const response = await backend.generateTicket(token, Number(selectedEventId));
			generatedCode = response.code;
			ticketMessage = `Ticket #${response.ticket.id} generated.`;
		} catch (error) {
			ticketMessage = error instanceof Error ? error.message : 'Ticket generation failed';
		}
	}

	async function validateTicket() {
		validationResult = null;
		validationMessage = '';
		try {
			validationResult = await backend.validateTicket(validationCode);
		} catch (error) {
			validationMessage = error instanceof Error ? error.message : 'Validation failed';
		}
	}

	async function checkInTicket() {
		checkInMessage = '';
		if (!token) {
			checkInMessage = 'Log in first.';
			return;
		}
		try {
			const response = await backend.checkInTicket(token, checkInCode);
			checkInMessage = `Ticket #${response.ticket.id} checked in.`;
		} catch (error) {
			checkInMessage = error instanceof Error ? error.message : 'Check-in failed';
		}
	}

	const selectedEvent = $derived(events.find((event) => String(event.id) === selectedEventId) ?? null);
</script>

<svelte:head>
	<title>Chime Frontend</title>
	<meta name="description" content="Frontend for the Chime backend" />
</svelte:head>

<div class="mx-auto flex min-h-screen max-w-7xl flex-col gap-8 px-4 py-8 md:px-8">
	<section class="grid gap-6 lg:grid-cols-[1.15fr_0.85fr]">
		<Card class="relative overflow-hidden">
			<div class="absolute inset-x-0 top-0 h-1 bg-gradient-to-r from-[var(--primary)] via-orange-300 to-amber-200"></div>
			<div class="flex flex-col gap-5">
				<div class="flex items-center gap-3 text-[var(--secondary-foreground)]">
					<div class="rounded-2xl bg-[var(--secondary)] p-3"><CalendarClock class="size-6" /></div>
					<div>
						<p class="text-xs uppercase tracking-[0.3em] text-[var(--muted-foreground)]">SvelteKit</p>
						<h1 class="text-4xl font-semibold tracking-tight">Chime control room</h1>
					</div>
				</div>
				<p class="max-w-2xl text-base leading-7 text-[var(--muted-foreground)]">
					A small UI for your Chi backend. Register, create events, mint tickets, validate codes,
					and check people in from one page.
				</p>
				<div class="flex flex-wrap gap-3 text-sm text-[var(--secondary-foreground)]">
					<span class="rounded-full bg-[var(--secondary)] px-3 py-1">API: {backend.baseUrl}</span>
					{#if currentUser}
						<span class="rounded-full bg-[var(--secondary)] px-3 py-1">{currentUser.email}</span>
					{/if}
				</div>
			</div>
		</Card>

		<Card>
			<div class="flex items-center gap-3">
				<div class="rounded-2xl bg-[var(--secondary)] p-3"><UserRound class="size-5" /></div>
				<div>
					<h2 class="text-xl font-semibold">Auth</h2>
					<p class="text-sm text-[var(--muted-foreground)]">Token is stored in local storage.</p>
				</div>
			</div>
			<div class="mt-5 grid gap-4">
				<div class="grid gap-2">
					<Label for="email">Email</Label>
					<Input id="email" bind:value={email} type="email" placeholder="me@example.com" />
				</div>
				<div class="grid gap-2">
					<Label for="password">Password</Label>
					<Input id="password" bind:value={password} type="password" placeholder="password123" />
				</div>
				<div class="flex flex-wrap gap-3">
					<Button disabled={busyAuth} onclick={register}>Register</Button>
					<Button variant="secondary" disabled={busyAuth} onclick={login}>Log in</Button>
					<Button variant="outline" onclick={logout}>Log out</Button>
				</div>
				{#if authMessage}
					<p class="text-sm text-[var(--secondary-foreground)]">{authMessage}</p>
				{/if}
			</div>
		</Card>
	</section>

	<section class="grid gap-6 xl:grid-cols-[1.1fr_0.9fr]">
		<Card>
			<div class="flex items-center gap-3">
				<div class="rounded-2xl bg-[var(--secondary)] p-3"><Ticket class="size-5" /></div>
				<div>
					<h2 class="text-xl font-semibold">Events and tickets</h2>
					<p class="text-sm text-[var(--muted-foreground)]">Create events and generate codes.</p>
				</div>
			</div>
			<div class="mt-6 grid gap-6 lg:grid-cols-[0.9fr_1.1fr]">
				<div class="grid gap-3">
					<h3 class="text-sm font-semibold uppercase tracking-[0.25em] text-[var(--muted-foreground)]">Events</h3>
					{#if busyEvents}
						<p class="text-sm text-[var(--muted-foreground)]">Loading events...</p>
					{:else if events.length === 0}
						<p class="text-sm text-[var(--muted-foreground)]">No events yet.</p>
					{:else}
						<div class="grid gap-3">
							{#each events as event}
								<button
									class={`rounded-2xl border p-4 text-left transition ${
										selectedEventId === String(event.id)
											? 'border-[var(--primary)] bg-[var(--secondary)]'
											: 'border-[var(--border)] bg-white/70 hover:bg-[var(--muted)]'
									}`}
									onclick={() => (selectedEventId = String(event.id))}
								>
									<p class="font-semibold">{event.name}</p>
									<p class="mt-1 text-xs text-[var(--muted-foreground)]">{new Date(event.starts).toLocaleString()}</p>
								</button>
							{/each}
						</div>
					{/if}
				</div>

				<div class="grid gap-6">
					<div class="grid gap-4">
						<h3 class="text-sm font-semibold uppercase tracking-[0.25em] text-[var(--muted-foreground)]">Create event</h3>
						<div class="grid gap-2">
							<Label for="event-name">Name</Label>
							<Input id="event-name" bind:value={eventName} placeholder="Launch party" />
						</div>
						<div class="grid gap-2">
							<Label for="event-starts">Starts</Label>
							<Input id="event-starts" bind:value={eventStarts} type="datetime-local" />
						</div>
						<div class="grid gap-2">
							<Label for="event-ends">Ends</Label>
							<Input id="event-ends" bind:value={eventEnds} type="datetime-local" />
						</div>
						<div class="flex gap-3">
							<Button disabled={busyEvents} onclick={createEvent}>Create event</Button>
							<Button variant="outline" disabled={busyEvents} onclick={loadEvents}>Refresh</Button>
						</div>
					</div>

					<div class="grid gap-4 rounded-[calc(var(--radius)+0.25rem)] bg-[var(--muted)] p-4">
						<h3 class="text-sm font-semibold uppercase tracking-[0.25em] text-[var(--muted-foreground)]">Generate ticket</h3>
						<p class="text-sm text-[var(--secondary-foreground)]">
							{#if selectedEvent}
								Using {selectedEvent.name} (#{selectedEvent.id})
							{:else}
								Pick an event first.
							{/if}
						</p>
						<Button onclick={generateTicket}>Generate</Button>
						{#if generatedCode}
							<div class="rounded-2xl bg-white/85 p-3 font-mono text-sm break-all">{generatedCode}</div>
						{/if}
					</div>
				</div>
			</div>

			{#if eventsMessage}<p class="mt-4 text-sm text-[var(--secondary-foreground)]">{eventsMessage}</p>{/if}
			{#if ticketMessage}<p class="mt-2 text-sm text-[var(--secondary-foreground)]">{ticketMessage}</p>{/if}
		</Card>

		<div class="grid gap-6">
			<Card>
				<div class="flex items-center gap-3">
					<div class="rounded-2xl bg-[var(--secondary)] p-3"><ScanQrCode class="size-5" /></div>
					<div>
						<h2 class="text-xl font-semibold">Validate ticket</h2>
						<p class="text-sm text-[var(--muted-foreground)]">Public endpoint.</p>
					</div>
				</div>
				<div class="mt-5 grid gap-4">
					<div class="grid gap-2">
						<Label for="validation-code">Code</Label>
						<Input id="validation-code" bind:value={validationCode} placeholder="Paste ticket code" />
					</div>
					<Button onclick={validateTicket}>Validate</Button>
					{#if validationResult}
						<div class="rounded-2xl bg-[var(--muted)] p-4 text-sm text-[var(--secondary-foreground)]">
							<p><strong>Status:</strong> {validationResult.status}</p>
							<p><strong>Valid:</strong> {validationResult.valid ? 'yes' : 'no'}</p>
							<p><strong>Ticket:</strong> {validationResult.ticket.id || 'n/a'}</p>
						</div>
					{/if}
					{#if validationMessage}<p class="text-sm text-[var(--secondary-foreground)]">{validationMessage}</p>{/if}
				</div>
			</Card>

			<Card>
				<div class="flex items-center gap-3">
					<div class="rounded-2xl bg-[var(--secondary)] p-3"><KeyRound class="size-5" /></div>
					<div>
						<h2 class="text-xl font-semibold">Check in ticket</h2>
						<p class="text-sm text-[var(--muted-foreground)]">Requires auth.</p>
					</div>
				</div>
				<div class="mt-5 grid gap-4">
					<div class="grid gap-2">
						<Label for="checkin-code">Code</Label>
						<Input id="checkin-code" bind:value={checkInCode} placeholder="Paste ticket code" />
					</div>
					<Button variant="secondary" onclick={checkInTicket}>Check in</Button>
					{#if checkInMessage}<p class="text-sm text-[var(--secondary-foreground)]">{checkInMessage}</p>{/if}
				</div>
			</Card>
		</div>
	</section>
</div>
