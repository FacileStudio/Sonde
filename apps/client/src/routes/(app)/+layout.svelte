<script lang="ts">
	import { onMount, setContext } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { Button, EmptyState, MobileNav, PageTransition, SideBar, Spinner, icons } from '@facile/muse';
	import { backend, type AuthUser } from '$lib/backend';
	import { setSessionLocale, t } from '$lib/i18n/index.svelte';

	let { children } = $props();

	let user = $state<AuthUser | null>(null);
	let checked = $state(false);
	let collapsed = $state(false);
	let unreachable = $state(false);
	let scroller = $state<HTMLElement | null>(null);

	function setUser(next: AuthUser) {
		user = next;
	}

	setContext('app', {
		get user() {
			return user;
		},
		setUser
	});

	function isActive(href: string) {
		return page.url.pathname === href || page.url.pathname.startsWith(href + '/');
	}

	/* Every /status-pages route on the API sits behind requireAdmin (internal/httpapi/router.go),
	   so the row is only a route a member can actually walk. The page itself refuses too, for
	   anyone arriving on a link. */
	const navPages = $derived([
		{ href: '/monitors', label: t('nav.monitors'), icon: icons.monitor, active: isActive('/monitors') },
		{ href: '/incidents', label: t('nav.incidents'), icon: icons.warning, active: isActive('/incidents') },
		...(user?.is_admin
			? [
					{
						href: '/status-pages',
						label: t('nav.statusPages'),
						icon: icons.globe,
						active: isActive('/status-pages')
					}
				]
			: [])
	]);

	/* The email is the fallback for a missing name, not the name itself. The avatar is
	   Registre's stable URL; muse's Avatar draws the initial when it 404s. */
	const navUser = $derived(
		user ? { name: user.name?.trim() || user.email, avatar: user.avatar_url || undefined } : undefined
	);
	const settingsActive = $derived(isActive('/settings'));

	onMount(async () => {
		try {
			user = (await backend.me()).user;
			setSessionLocale(user.locale);
			checked = true;
		} catch (err) {
			/* Only a refused credential logs someone out. A 500 or a dropped connection means
			   the server is having a bad day, and bouncing to /login loses whatever they were
			   doing without fixing anything. apiFetch already sent a 401 to /login with the
			   current path attached, so this only has to stop rendering. */
			const status = (err as { status?: number }).status;
			if (status === 401) return;
			if (status === 403) {
				goto(`/login?next=${encodeURIComponent(page.url.pathname)}`);
				return;
			}
			unreachable = true;
		}
	});

	/* The scroller is the element that scrolls, so window.scrollTo does nothing here: without
	   this a navigation lands halfway down the new page. */
	$effect(() => {
		void page.url.pathname;
		scroller?.scrollTo({ top: 0 });
	});
</script>

<div class="flex h-dvh w-full overflow-hidden bg-fc-page">
	<div class="hidden h-full shrink-0 p-3 md:block">
		<SideBar
			class="h-full"
			icon="solar:radar-2-bold-duotone"
			title="Sonde"
			bind:collapsed
			pages={navPages}
			user={navUser}
			userHref="/settings"
			userActive={settingsActive}
		/>
	</div>
	<main
		bind:this={scroller}
		class="min-w-0 flex-1 overflow-auto overscroll-contain pb-28 md:pb-0"
	>
		<div class="mx-auto flex w-full max-w-fc-xl flex-col gap-10 px-4 py-8 sm:px-6 md:px-10 md:py-10">
			{#if checked}
				<PageTransition key={page.url.pathname}>
					{@render children()}
				</PageTransition>
			{:else if unreachable}
				<EmptyState
					icon={icons.error}
					title={t('shell.unreachable.title')}
					description={t('shell.unreachable.description')}
				>
					<Button icon={icons.refresh} onclick={() => location.reload()}>{t('shell.tryAgain')}</Button>
				</EmptyState>
			{:else}
				<div class="flex min-h-[50dvh] items-center justify-center">
					<Spinner />
				</div>
			{/if}
		</div>
	</main>
	<MobileNav
		items={navPages}
		user={navUser}
		profileHref="/settings"
		profileActive={settingsActive}
	/>
</div>
