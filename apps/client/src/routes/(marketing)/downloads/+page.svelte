<script lang="ts">
	import { DownloadSection, LandingFooter, LandingHeader, SignInWithFacile } from '@facile/muse';
	import { ssoLoginUrl } from '$lib/backend';
	import { CHECKSUMS_URL, INSTALL_METHODS, RELEASES_URL } from '$lib/downloads';
	import { rememberLocale, t } from '$lib/i18n/index.svelte';

	const ICON = 'solar:radar-bold-duotone';
</script>

<svelte:head>
	<title>{t('downloads.title')} · Sonde</title>
	<meta name="description" content={t('downloads.intro')} />
</svelte:head>

<div class="flex min-h-dvh flex-col overflow-x-clip bg-fc-page text-fc-fg">
	<LandingHeader appName="Sonde" icon={ICON} homeHref="/" skipHref="#main">
		{#snippet actions()}
			<SignInWithFacile href={ssoLoginUrl} />
		{/snippet}
	</LandingHeader>

	<main id="main" tabindex="-1" class="mx-auto flex w-full max-w-5xl flex-1 flex-col gap-8 px-4 py-16 sm:px-6 sm:py-20">
		<div class="text-center">
			<h1 class="font-fc-title text-fc-3xl font-bold tracking-tight">{t('downloads.title')}</h1>
			<p class="mx-auto mt-3 max-w-md text-fc-sm text-fc-fg-muted">{t('downloads.intro')}</p>
		</div>
		<DownloadSection commands={INSTALL_METHODS} releasesHref={RELEASES_URL} checksumsHref={CHECKSUMS_URL} headingLevel={2} />
	</main>

	<LandingFooter appName="Sonde" icon={ICON} tagline={t('landing.tagline')} legal={{ onLocaleChange: rememberLocale }} />
</div>
