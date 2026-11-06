<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { Alert, AuthScreen } from '@facile/muse';
	import { backend, ssoLoginUrl } from '$lib/backend';
	import { rememberLocale, t } from '$lib/i18n/index.svelte';

	let failed = $state(false);

	/*
	 * `next` arrives in the address bar, so it is attacker-controlled: anything but a
	 * same-origin absolute path is an open redirect. `//evil.com` and `/\evil.com` are
	 * protocol-relative URLs the browser resolves against another host, and a leading `/`
	 * already rules out `https:` and `javascript:`. Bouncing back to /login would loop.
	 */
	function safeNext(raw: string | null): string {
		if (!raw || !raw.startsWith('/')) return '/monitors';
		if (raw.startsWith('//') || raw.startsWith('/\\')) return '/monitors';
		if (raw === '/login' || raw.startsWith('/login/') || raw.startsWith('/login?')) {
			return '/monitors';
		}
		return raw;
	}

	onMount(async () => {
		failed = page.url.searchParams.has('error');
		try {
			await backend.probeSession();
			goto(safeNext(page.url.searchParams.get('next')));
		} catch {
			/* A 401 here is the expected answer for a signed-out visitor. */
		}
	});
</script>

<svelte:head>
	<title>{t('login.title')}</title>
</svelte:head>

<AuthScreen appName="Sonde" href={ssoLoginUrl} onLocaleChange={rememberLocale}>
	{#snippet logo()}
		<iconify-icon icon="solar:radar-2-bold-duotone" width="40" height="40" class="block text-fc-fg"></iconify-icon>
	{/snippet}
	{#snippet tagline()}
		{t('login.tagline')}
	{/snippet}
	{#snippet notice()}
		{#if failed}
			<Alert tone="danger">{t('login.failed')}</Alert>
		{/if}
	{/snippet}
</AuthScreen>
