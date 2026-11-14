<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { Button, DownloadSection, LandingFooter, LandingHeader, SignInWithFacile, icons } from '@facile/muse';
	import LandingOffers from '$lib/components/LandingOffers.svelte';
	import MonitorsDemo from '$lib/components/landing/MonitorsDemo.svelte';
	import { backend, ssoLoginUrl } from '$lib/backend';
	import { CHECKSUMS_URL, INSTALL_METHODS, RELEASES_URL } from '$lib/downloads';
	import { rememberLocale, t } from '$lib/i18n/index.svelte';

	const ICON = 'solar:radar-bold-duotone';
	const FEATURES = [
		{ icon: 'solar:monitor-linear', id: 'probes' },
		{ icon: 'solar:chart-2-linear', id: 'uptime' },
		{ icon: 'solar:global-linear', id: 'status' },
		{ icon: 'solar:danger-triangle-linear', id: 'alerts' }
	] as const;

	let checking = $state(true);

	const links = $derived([
		{ href: '#features', label: t('landing.nav.features') },
		{ href: '#download', label: t('landing.nav.download') },
		{ href: '#pricing', label: t('landing.nav.pricing') }
	]);

	const columns = $derived([
		{
			title: t('landing.column.product'),
			links: [
				{ href: '#features', label: t('landing.nav.features') },
				{ href: '/downloads', label: t('downloads.title') },
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
	<div class="flex min-h-dvh flex-col overflow-x-clip bg-fc-page text-fc-fg">
		<LandingHeader appName="Sonde" icon={ICON} {links} skipHref="#main">
			{#snippet actions()}
				<SignInWithFacile href={ssoLoginUrl} />
			{/snippet}
		</LandingHeader>

		<main id="main" tabindex="-1" class="flex-1">
			<section class="mx-auto w-full max-w-5xl px-4 pt-16 pb-10 sm:px-6 sm:pt-24">
				<h1 class="font-fc-title text-fc-3xl leading-tight font-bold tracking-tight sm:text-6xl">
					{t('landing.hero.line1')}<br /><span class="text-fc-fg-muted">{t('landing.hero.line2')}</span>
				</h1>
				<p class="mt-6 max-w-xl text-fc-md text-fc-fg-muted">{t('landing.hero.text')}</p>
				<div class="mt-8 flex flex-wrap items-center gap-3">
					<SignInWithFacile href={ssoLoginUrl} size="lg" />
					<Button href="#download" variant="secondary" size="lg" icon={icons.download}>
						{t('downloads.cta')}
					</Button>
				</div>
			</section>

			<section class="mx-auto w-full max-w-5xl px-4 pb-8 sm:px-6">
				<MonitorsDemo />
			</section>

			<section id="features" class="mx-auto w-full max-w-5xl scroll-mt-20 px-4 py-12 sm:px-6">
				<ul class="grid gap-4 sm:grid-cols-2">
					{#each FEATURES as feature (feature.id)}
						<li class="rounded-fc-lg bg-fc-component p-6">
							<div class="mb-4 flex size-10 items-center justify-center rounded-fc-md bg-fc-surface" aria-hidden="true">
								<iconify-icon icon={feature.icon} width="20" height="20" class="block shrink-0"></iconify-icon>
							</div>
							<h2 class="text-fc-lg font-bold">{t(`landing.feature.${feature.id}.title`)}</h2>
							<p class="mt-1.5 text-fc-sm text-fc-fg-muted">{t(`landing.feature.${feature.id}.text`)}</p>
						</li>
					{/each}
				</ul>
			</section>

			<section id="download" class="mx-auto flex w-full max-w-5xl scroll-mt-20 flex-col gap-8 px-4 py-16 sm:px-6">
				<div class="text-center">
					<h2 class="font-fc-title text-fc-2xl font-bold tracking-tight">{t('downloads.heading')}</h2>
					<p class="mx-auto mt-3 max-w-md text-fc-sm text-fc-fg-muted">{t('downloads.intro')}</p>
				</div>
				<DownloadSection
					mode="device"
					allHref="/downloads"
					commands={INSTALL_METHODS}
					releasesHref={RELEASES_URL}
					checksumsHref={CHECKSUMS_URL}
					headingLevel={3}
				/>
			</section>

			<div id="pricing" class="scroll-mt-20">
				<LandingOffers />
			</div>

			<section class="mx-auto w-full max-w-5xl px-4 py-16 text-center sm:px-6">
				<h2 class="font-fc-title text-fc-2xl font-bold tracking-tight">{t('landing.cta.title')}</h2>
				<p class="mt-4 text-fc-sm text-fc-fg-muted">{t('landing.cta.text')}</p>
				<div class="mt-8 flex justify-center">
					<SignInWithFacile href={ssoLoginUrl} size="lg" />
				</div>
			</section>
		</main>

		<LandingFooter
			appName="Sonde"
			icon={ICON}
			tagline={t('landing.tagline')}
			{columns}
			legal={{ onLocaleChange: rememberLocale }}
		/>
	</div>
{/if}
