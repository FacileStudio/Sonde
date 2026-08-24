<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { Button, Card, Field, Input, Alert, Spinner } from '@facile/muse';
	import { backend, ssoLoginUrl } from '$lib/backend';

	let email = $state('');
	let password = $state('');
	let name = $state('');
	let mode = $state<'login' | 'register'>('login');
	let ssoOnly = $state(false);
	let ssoEnabled = $state(false);
	let allowRegistration = $state(false);
	let busy = $state(false);
	let error = $state('');

	onMount(async () => {
		const refused = new URLSearchParams(location.search).get('error');
		if (refused) error = refused;

		try {
			await backend.me();
			goto('/');
		} catch {
			try {
				const cfg = await backend.authConfig();
				ssoOnly = cfg.sso_only;
				ssoEnabled = cfg.oidc_enabled;
				allowRegistration = cfg.allow_registration;
			} catch {
				ssoOnly = false;
			}
		}
	});

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (busy) return;
		busy = true;
		error = '';
		try {
			if (mode === 'register') {
				await fetch('/api/auth/register', {
					method: 'POST',
					headers: { 'Content-Type': 'application/json', 'X-Facile-CSRF': '1' },
					body: JSON.stringify({ email, name, password })
				});
			}
			const response = await fetch('/api/auth/login', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ email, password })
			});
			if (!response.ok) {
				const payload = (await response.json().catch(() => undefined)) as
					| { error?: { message?: string } }
					| undefined;
				throw new Error(payload?.error?.message ?? 'Authentication failed.');
			}
			goto('/');
		} catch (err) {
			error = err instanceof Error ? err.message : 'Authentication failed.';
			busy = false;
		}
	}
</script>

<svelte:head>
	<title>Log in — Sonde</title>
</svelte:head>

<div class="flex min-h-dvh items-center justify-center bg-fc-page px-4">
	<div class="w-full max-w-sm">
		<div class="mb-8 text-center">
			<h1 class="text-fc-xl font-semibold text-fc-fg">Sonde</h1>
			<p class="text-fc-sm text-fc-fg-muted">Uptime monitoring for the Facile Suite</p>
		</div>

		<Card class="p-6">
			{#if error}
				<Alert tone="danger" class="mb-4">{error}</Alert>
			{/if}

			{#if ssoEnabled}
				<a href={ssoLoginUrl} class="block">
					<Button type="button" class="w-full" disabled={busy}>Sign in with SSO</Button>
				</a>
			{/if}

			{#if !ssoOnly}
				{#if ssoEnabled}
					<div class="my-4 flex items-center gap-3">
						<div class="h-px flex-1 bg-fc-border"></div>
						<span class="text-fc-xs text-fc-fg-muted">or</span>
						<div class="h-px flex-1 bg-fc-border"></div>
					</div>
				{/if}

				<form onsubmit={submit} class="flex flex-col gap-4">
					{#if mode === 'register'}
						<Field label="Name">
							{#snippet children({ id })}
								<Input {id} bind:value={name} autocomplete="name" required />
							{/snippet}
						</Field>
					{/if}
					<Field label="Email">
						{#snippet children({ id })}
							<Input
								{id}
								type="email"
								bind:value={email}
								autocomplete="email"
								required
							/>
						{/snippet}
					</Field>
					<Field label="Password">
						{#snippet children({ id })}
							<Input
								{id}
								type="password"
								bind:value={password}
								autocomplete={mode === 'register' ? 'new-password' : 'current-password'}
								required
							/>
						{/snippet}
					</Field>
					<Button type="submit" disabled={busy}>
						{#if busy}
							<Spinner class="mr-2" />
						{/if}
						{mode === 'register' ? 'Create account' : 'Sign in'}
					</Button>
				</form>

				{#if allowRegistration}
					<button
						type="button"
						class="mt-4 w-full text-center text-fc-sm text-fc-fg-muted hover:text-fc-fg"
						onclick={() => {
							mode = mode === 'login' ? 'register' : 'login';
							error = '';
						}}
					>
						{mode === 'login' ? 'No account yet? Register' : 'Already registered? Sign in'}
					</button>
				{/if}
			{/if}
		</Card>
	</div>
</div>
