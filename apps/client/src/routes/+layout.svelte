<script lang="ts">
	import '../app.css';
	import { browser } from '$app/environment';
	import { Toaster, provideLocale } from '@facile/muse';
	import { bindLocale, localeSource } from '$lib/i18n/index.svelte';
	import { restoreTheme } from '$lib/theme.svelte';
	import mdi from '$lib/icons/mdi.json';
	import solar from '$lib/icons/solar.json';

	let { children } = $props();

	const i18n = provideLocale(localeSource);
	bindLocale(() => i18n.locale);

	if (browser) {
		restoreTheme();

		/*
		 * Registering the custom element is what makes muse's chrome visible at all — its
		 * components render raw <iconify-icon> tags that stay inert until this module loads.
		 * Registering the collections in the same breath is what keeps the glyphs local: the
		 * element resolves anything it does not know by calling api.iconify.design from the
		 * user's browser, so an unbundled icon is a third-party runtime dependency in an app
		 * whose whole promise is not having one.
		 *
		 * Regenerate with `bun run icons` after adding an icon or bumping @facile/muse.
		 */
		void import('iconify-icon').then(({ addCollection }) => {
			addCollection(solar);
			addCollection(mdi);
		});
	}
</script>

<svelte:head>
	<link rel="icon" href="/favicon.ico" type="image/x-icon" />
</svelte:head>

<!-- One Toaster for the whole app, outside the router: mounted inside a page, a navigation
     unmounts a toast mid-flight. The bottom padding clears MobileNav on phones. -->
<Toaster class="pb-28 md:pb-6" />
{@render children()}
