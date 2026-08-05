<script lang="ts">
	import { goto } from '$app/navigation';
	import { backend, tokenStorageKey } from '$lib/backend';

	let email = $state('');
	let password = $state('');
	let message = $state('');
	let busy = $state(false);

	async function handleLogin(event: SubmitEvent) {
		event.preventDefault();
		busy = true;
		message = '';
		try {
			const response = await backend.login(email, password);
			localStorage.setItem(tokenStorageKey, response.token);
			await goto('/');
		} catch (error) {
			message = error instanceof Error ? error.message : 'Authentication failed.';
			busy = false;
		}
	}

	const inputClass =
		'flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50';
	const labelClass = 'text-sm font-medium leading-none';
	const primaryButtonClass =
		'inline-flex h-10 w-full items-center justify-center rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:pointer-events-none disabled:opacity-50';
</script>

<svelte:head>
	<title>Log in — App</title>
</svelte:head>

<div class="flex min-h-screen">
	<div class="hidden lg:flex lg:w-1/2 flex-col bg-black px-12 py-10">
		<a href="/" class="flex items-center gap-3 mb-auto">
			<span class="text-xl font-bold font-heading tracking-tight text-white">App</span>
		</a>
		<div class="mb-auto">
			<h2 class="text-4xl font-bold font-heading text-white leading-tight tracking-tight">
				Your app.<br />Your server.
			</h2>
			<p class="mt-4 text-sm text-white/50 max-w-xs leading-relaxed">
				A self-hosted Go + SvelteKit starter by Facile.
			</p>
		</div>
		<p class="text-xs text-white/30">© {new Date().getFullYear()} App by Facile.</p>
	</div>

	<div class="flex w-full lg:w-1/2 flex-col items-center justify-center px-8 py-12 bg-background">
		<div class="w-full max-w-sm">
			<div class="mb-8">
				<h1 class="text-2xl font-bold font-heading tracking-tight text-foreground">Welcome back</h1>
				<p class="mt-1.5 text-sm text-muted-foreground">Log in to your App account.</p>
			</div>

			<form class="space-y-4" onsubmit={handleLogin}>
				<div class="space-y-1.5">
					<label for="email" class={labelClass}>Email</label>
					<input
						id="email"
						type="email"
						bind:value={email}
						placeholder="you@example.com"
						autocomplete="email"
						required
						disabled={busy}
						class={inputClass}
					/>
				</div>
				<div class="space-y-1.5">
					<label for="password" class={labelClass}>Password</label>
					<input
						id="password"
						type="password"
						bind:value={password}
						placeholder="••••••••"
						autocomplete="current-password"
						required
						disabled={busy}
						class={inputClass}
					/>
				</div>

				{#if message}
					<p
						class="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive"
					>
						{message}
					</p>
				{/if}

				<button type="submit" disabled={busy} class={primaryButtonClass}>
					{busy ? 'Logging in…' : 'Log in'}
				</button>
			</form>
		</div>
	</div>
</div>
