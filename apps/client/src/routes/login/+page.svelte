<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { backend, ssoLoginUrl } from '$lib/backend';

	let email = $state('');
	let password = $state('');
	let name = $state('');
	let mode = $state<'login' | 'register'>('login');
	let ssoOnly = $state(false);
	let ssoEnabled = $state(false);
	let allowRegistration = $state(false);
	let configLoaded = $state(false);
	let busy = $state(false);
	let error = $state('');

	onMount(async () => {
		const refused = new URLSearchParams(location.search).get('error');
		if (refused) error = refused;
		if (new URLSearchParams(location.search).get('tab') === 'register') mode = 'register';

		try {
			await backend.me();
			goto('/monitors');
			return;
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
		configLoaded = true;
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
			goto('/monitors');
		} catch (err) {
			error = err instanceof Error ? err.message : 'Authentication failed.';
			busy = false;
		}
	}
</script>

<svelte:head>
	<title>{!ssoOnly && mode === 'register' ? 'Create account' : 'Log in'} — Sonde</title>
</svelte:head>

<div class="flex min-h-screen">
	<div class="hidden flex-col border-r border-fc-border bg-black px-12 py-10 lg:flex lg:w-1/2">
		<a href="/" class="mb-auto flex items-center gap-3">
			<iconify-icon
				icon="solar:radar-2-bold-duotone"
				width="24"
				height="24"
				class="block shrink-0 text-white"
			></iconify-icon>
			<span class="font-fc-title text-fc-xl font-semibold tracking-tight text-white">Sonde</span>
		</a>

		<div class="mb-auto">
			<h2 class="font-fc-title text-fc-3xl leading-tight font-semibold tracking-tight text-white">
				Is it up?<br />Know in a minute.
			</h2>
			<p class="mt-4 max-w-xs text-fc-sm leading-relaxed text-white/50">
				Self-hosted uptime monitoring for your services, your cron jobs and your clients' peace of mind.
			</p>
		</div>

		<p class="text-fc-xs text-white/30">
			© {new Date().getFullYear()} Sonde by Facile.
		</p>
	</div>

	<div class="flex w-full flex-col items-center justify-center bg-fc-page px-8 py-12 lg:w-1/2">
		<div class="mb-6 flex items-center gap-3 lg:hidden">
			<iconify-icon icon="solar:radar-2-bold-duotone" width="22" height="22" class="block shrink-0 text-fc-fg"></iconify-icon>
			<a href="/" class="font-fc-title text-lg font-semibold tracking-tight text-fc-fg">Sonde</a>
		</div>

		<div class="w-full max-w-sm">
			<div class="mb-8">
				<h1 class="font-fc-title text-fc-2xl font-semibold tracking-tight text-fc-fg">
					{!ssoOnly && mode === 'register' ? 'Create account' : 'Welcome back'}
				</h1>
				<p class="mt-1.5 text-fc-sm text-fc-fg-muted">
					{!ssoOnly && mode === 'register'
						? 'Start watching what keeps you up at night.'
						: ssoOnly
							? 'Sign in with your Facile account to access Sonde.'
							: 'Log in to your Sonde account.'}
				</p>
			</div>

			{#if !configLoaded}
				<div class="h-40"></div>
			{:else}
				{#if error}
					<p role="alert" class="mb-4 rounded-md bg-fc-danger/10 px-3 py-2 text-fc-sm text-fc-danger">
						{error}
					</p>
				{/if}

				{#if ssoEnabled}
					<a href={ssoLoginUrl} data-sveltekit-reload
						class="inline-flex h-10 w-full items-center justify-center rounded-md bg-fc-accent px-4 text-sm font-medium text-fc-accent-fg transition-opacity hover:opacity-90">
						<iconify-icon icon="solar:key-minimalistic-square-bold-duotone" width="18" height="18" class="mr-2"></iconify-icon>
						Sign in with Facile
					</a>
					{#if !ssoOnly}
						<div class="my-6 flex items-center gap-3">
							<span class="h-px flex-1 bg-fc-border"></span>
							<span class="text-xs uppercase tracking-wide text-fc-fg-muted">or</span>
							<span class="h-px flex-1 bg-fc-border"></span>
						</div>
					{/if}
				{/if}

				{#if !ssoOnly}
					<div class="mb-6 flex gap-1 rounded-lg border border-fc-border bg-fc-surface p-1" role="tablist">
						<button
							type="button"
							role="tab"
							aria-selected={mode === 'login'}
							class="flex-1 rounded-md py-1.5 text-sm font-medium transition-colors {mode === 'login'
								? 'bg-fc-page text-fc-fg shadow-sm'
								: 'text-fc-fg-muted hover:text-fc-fg'}"
							onclick={() => {
								mode = 'login';
								error = '';
							}}>Log in</button
						>
						{#if allowRegistration}
							<button
								type="button"
								role="tab"
								aria-selected={mode === 'register'}
								class="flex-1 rounded-md py-1.5 text-sm font-medium transition-colors {mode === 'register'
									? 'bg-fc-page text-fc-fg shadow-sm'
									: 'text-fc-fg-muted hover:text-fc-fg'}"
								onclick={() => {
									mode = 'register';
									error = '';
								}}>Register</button
							>
						{/if}
					</div>

					<form onsubmit={submit} class="flex flex-col gap-4">
						{#if mode === 'register'}
							<label class="flex flex-col gap-1.5">
								<span class="text-sm font-medium text-fc-fg">Name</span>
								<input
									type="text"
									bind:value={name}
									autocomplete="name"
									required
									class="flex h-10 w-full rounded-md border border-fc-border bg-fc-page px-3 py-2 text-sm text-fc-fg placeholder:text-fc-fg-muted focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring"
								/>
							</label>
						{/if}
						<label class="flex flex-col gap-1.5">
							<span class="text-sm font-medium text-fc-fg">Email</span>
							<input
								type="email"
								bind:value={email}
								autocomplete="email"
								required
								class="flex h-10 w-full rounded-md border border-fc-border bg-fc-page px-3 py-2 text-sm text-fc-fg placeholder:text-fc-fg-muted focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring"
							/>
						</label>
						<label class="flex flex-col gap-1.5">
							<span class="text-sm font-medium text-fc-fg">Password</span>
							<input
								type="password"
								bind:value={password}
								autocomplete={mode === 'register' ? 'new-password' : 'current-password'}
								required
								class="flex h-10 w-full rounded-md border border-fc-border bg-fc-page px-3 py-2 text-sm text-fc-fg placeholder:text-fc-fg-muted focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-fc-ring"
							/>
						</label>
						<button
							type="submit"
							disabled={busy}
							class="inline-flex h-10 w-full items-center justify-center rounded-md bg-fc-accent px-4 text-sm font-medium text-fc-accent-fg transition-opacity hover:opacity-90 disabled:pointer-events-none disabled:opacity-50"
						>
							{busy ? 'One moment…' : mode === 'register' ? 'Create account' : 'Log in'}
						</button>
					</form>
				{/if}
			{/if}
		</div>
	</div>
</div>
