<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { LandingFooter, LandingHeader, SignInWithFacile } from '@facile/muse';
	import LandingOffers from '$lib/components/LandingOffers.svelte';
	import { backend, ssoLoginUrl } from '$lib/backend';
	import { rememberLocale, t } from '$lib/i18n/index.svelte';

	const ICON = 'solar:radar-2-bold-duotone';

	let checking = $state(true);

	const features = $derived([
		{ icon: 'solar:monitor-linear', title: t('landing.feature.probes.title'), text: t('landing.feature.probes.text') },
		{ icon: 'solar:chart-2-linear', title: t('landing.feature.uptime.title'), text: t('landing.feature.uptime.text') },
		{ icon: 'solar:global-linear', title: t('landing.feature.status.title'), text: t('landing.feature.status.text') },
		{ icon: 'solar:danger-triangle-linear', title: t('landing.feature.alerts.title'), text: t('landing.feature.alerts.text') }
	]);

	const links = $derived([
		{ href: '#features', label: t('landing.nav.features') },
		{ href: '#pricing', label: t('landing.nav.pricing') }
	]);

	const columns = $derived([
		{
			title: t('landing.column.product'),
			links: [
				{ href: '#features', label: t('landing.nav.features') },
				{ href: '#pricing', label: t('landing.nav.pricing') },
				{ href: '/docs', label: t('landing.link.api') }
			]
		},
		{
			title: t('landing.column.facile'),
			links: [{ href: 'https://facile.studio', label: 'Facile Studio', external: true }]
		}
	]);

	onMount(async () => {
		try {
			await backend.probeSession();
			goto('/monitors');
		} catch {
			checking = false;
		}
	});
</script>

<svelte:head>
	<title>{t('landing.title')}</title>
	<meta name="description" content={t('landing.description')} />
</svelte:head>

{#if checking}
	<div class="flex min-h-dvh items-center justify-center bg-fc-page"><span class="text-fc-fg-muted">…</span></div>
{:else}
	<div class="flex min-h-dvh flex-col bg-fc-page text-fc-fg">
		<LandingHeader appName="Sonde" icon={ICON} {links} skipHref="#main">
			{#snippet actions()}
				<SignInWithFacile href={ssoLoginUrl} />
			{/snippet}
		</LandingHeader>

		<main id="main" tabindex="-1" class="mx-auto flex w-full max-w-fc-xl flex-1 flex-col justify-center px-4 py-16 sm:px-6 md:px-10">
			<h1 class="font-fc-title text-4xl leading-tight font-semibold tracking-tight sm:text-6xl">
				{t('landing.hero.line1')}<br /><span class="text-fc-fg-muted">{t('landing.hero.line2')}</span>
			</h1>
			<p class="mt-6 max-w-xl text-base leading-relaxed text-fc-fg-muted">{t('landing.hero.text')}</p>
			<div class="mt-8">
				<SignInWithFacile href={ssoLoginUrl} />
			</div>

			<div id="features" class="mt-20 grid grid-cols-1 gap-8 scroll-mt-24 sm:grid-cols-2">
				{#each features as feature (feature.title)}
					<div>
						<iconify-icon icon={feature.icon} width="22" height="22" class="mb-3 block text-fc-fg-muted"></iconify-icon>
						<h2 class="text-sm font-semibold">{feature.title}</h2>
						<p class="mt-1.5 text-sm leading-relaxed text-fc-fg-muted">{feature.text}</p>
					</div>
				{/each}
			</div>
		</main>

		<div id="pricing" class="scroll-mt-24">
			<LandingOffers />
		</div>

		<LandingFooter
			appName="Sonde"
			icon={ICON}
			tagline={t('landing.tagline')}
			{columns}
			legal={{ onLocaleChange: rememberLocale }}
		/>
	</div>
{/if}
