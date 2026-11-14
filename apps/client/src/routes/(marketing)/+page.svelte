<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import {
		FeatureSplit,
		ForDevelopers,
		LANDING_HEADING,
		LandingCta,
		LandingFooter,
		LandingHeader,
		PricingPlans,
		SignInWithFacile,
		WorksWith,
		type DeveloperRow,
		type FeatureItem,
		type WorksWithApp
	} from '@facile/muse';
	import CronTile from '$lib/components/landing/CronTile.svelte';
	import MonitorsDemo from '$lib/components/landing/MonitorsDemo.svelte';
	import SiteDownTile from '$lib/components/landing/SiteDownTile.svelte';
	import StatusPageTile from '$lib/components/landing/StatusPageTile.svelte';
	import { backend, ssoLoginUrl } from '$lib/backend';
	import { INSTALL_METHODS } from '$lib/downloads';
	import { rememberLocale, t } from '$lib/i18n/index.svelte';

	const ICON = 'solar:radar-bold-duotone';
	const WIDTH = 'mx-auto w-full max-w-5xl px-4 sm:px-6';
	const SELF_HOST = 'https://suite.facile.studio/auto-hebergement';

	let checking = $state(true);

	const links = $derived([
		{ href: '#features', label: t('landing.nav.features') },
		{ href: '#pricing', label: t('landing.nav.pricing') },
		{ href: '#developers', label: t('landing.nav.developers') }
	]);

	const columns = $derived([
		{
			title: t('landing.column.product'),
			links: [
				{ href: '#features', label: t('landing.nav.features') },
				{ href: '#pricing', label: t('landing.nav.pricing') },
				{ href: '/downloads', label: t('downloads.title') },
				{ href: '/docs', label: t('landing.link.api') }
			]
		},
		{
			title: t('landing.column.facile'),
			links: [{ href: 'https://facile.studio', label: 'Facile Studio', external: true }]
		}
	]);

	const FEATURES = [
		{ id: 'probes', icon: 'solar:monitor-linear' },
		{ id: 'uptime', icon: 'solar:chart-2-linear' },
		{ id: 'incidents', icon: 'solar:danger-triangle-linear' },
		{ id: 'webhooks', icon: 'solar:link-linear' }
	] as const;

	const features: FeatureItem[] = $derived(
		FEATURES.map(({ id, icon }) => ({ icon, title: t(`landing.feature.${id}.title`), text: t(`landing.feature.${id}.text`) }))
	);

	const apps: WorksWithApp[] = $derived([
		{ name: 'Antenne', icon: 'solar:station-bold-duotone', line: t('landing.with.antenne'), href: 'https://antenne.facile.studio' }
	]);

	const developer: DeveloperRow[] = $derived([
		{ kind: 'cli', text: t('landing.dev.cli'), command: INSTALL_METHODS[0].command, href: '/downloads' },
		{ kind: 'api', text: t('landing.dev.api'), href: '/docs' },
		{ kind: 'source', text: t('landing.dev.source'), href: 'https://github.com/FacileStudio/Sonde' }
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
				<SignInWithFacile href={ssoLoginUrl} size="sm" />
			{/snippet}
		</LandingHeader>

		<main id="main" tabindex="-1" class="flex-1">
			<section class="{WIDTH} pt-16 pb-10 sm:pt-24">
				<h1 class="font-fc-title text-fc-3xl leading-tight font-bold tracking-tight sm:text-6xl">
					{t('landing.hero.line1')}<br /><span class="text-fc-fg-muted">{t('landing.hero.line2')}</span>
				</h1>
				<p class="mt-6 max-w-xl text-fc-md text-fc-fg-muted">{t('landing.hero.text')}</p>
				<LandingCta class="mt-8" signInHref={ssoLoginUrl} />
				<p class="mt-4 text-fc-sm text-fc-fg-muted">{t('landing.hero.note')}</p>
			</section>

			<section class="{WIDTH} pb-8">
				<MonitorsDemo />
			</section>

			<section id="features" class="{WIDTH} scroll-mt-20 py-16 sm:py-20">
				<h2 class={LANDING_HEADING}>{t('landing.cards.title')}</h2>
				<div class="mt-10 grid gap-4 md:grid-cols-3">
					<SiteDownTile />
					<StatusPageTile />
					<CronTile />
				</div>
			</section>

			<FeatureSplit
				class="{WIDTH} py-16 sm:py-20"
				title={t('landing.features.title')}
				description={t('landing.features.line')}
				{features}
			/>

			<WorksWith class="{WIDTH} py-16 sm:py-20" {apps} />

			<PricingPlans
				id="pricing"
				class="{WIDTH} scroll-mt-20 py-16 sm:py-20"
				title={t('landing.pricing')}
				headingClass={LANDING_HEADING}
				highlightOnly
				notes={{ '*': t('landing.plan.included') }}
				recommended="pro"
				suiteHref="https://suite.facile.studio/#tarifs"
			/>

			<div id="developers" class="{WIDTH} scroll-mt-20 py-16 sm:py-20">
				<ForDevelopers rows={developer} description={t('landing.dev.line')} selfHostHref={SELF_HOST} />
			</div>

			<section class="{WIDTH} py-16 text-center sm:py-24">
				<h2 class={LANDING_HEADING}>{t('landing.cta.title')}</h2>
				<p class="mt-3 text-fc-sm text-fc-fg-muted">{t('landing.cta.text')}</p>
				<LandingCta class="mt-8" signInHref={ssoLoginUrl} align="center" />
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
