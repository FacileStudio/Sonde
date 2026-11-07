<script lang="ts">
	import { getContext } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import {
		ConfirmModal,
		PreferencesSection,
		ProfileSection,
		SettingsLayout,
		icons
	} from '@facile/muse';
	import { backend, type AuthUser } from '$lib/backend';
	import { rememberLocale, t } from '$lib/i18n/index.svelte';
	import { setTheme, theme, type ThemeMode } from '$lib/theme.svelte';
	import WebhookList from '$lib/components/WebhookList.svelte';
	import BusSection from '$lib/components/BusSection.svelte';
	import AdvancedSection from '$lib/components/AdvancedSection.svelte';

	const app = getContext<{ user: AuthUser | null }>('app');

	const SECTIONS = [
		{ id: 'profile', icon: icons.userCircle },
		{ id: 'preferences', icon: icons.palette },
		{ id: 'notifications', icon: icons.notification },
		{ id: 'bus', icon: icons.plug },
		{ id: 'advanced', icon: icons.server }
	] as const;

	let clearingLocal = $state(false);

	const sections = $derived(
		SECTIONS.map((s) => ({ ...s, label: t(`settings.section.${s.id}`), href: `/settings/${s.id}` }))
	);
	const user = $derived(app.user);

	async function logout() {
		await backend.logout().catch(() => undefined);
		goto('/login');
	}

	function clearLocalData() {
		localStorage.removeItem('sonde-theme');
		logout();
	}
</script>

<svelte:head>
	<title>{t('nav.settings')} · Sonde</title>
</svelte:head>

<SettingsLayout
	{sections}
	active={page.params.section}
	backHref="/settings"
	title={t('nav.settings')}
	description={t('settings.description')}
>
	{#snippet children(id)}
		{#if id === 'profile'}
			<ProfileSection
				name={user?.name?.trim() || user?.email || ''}
				email={user?.email}
				avatar={user?.avatar_url || undefined}
				role={user?.is_admin ? t('settings.role.admin') : t('settings.role.member')}
			/>
		{:else if id === 'preferences'}
			<PreferencesSection
				theme={theme.mode}
				onThemeChange={(next) => setTheme(next as ThemeMode)}
				onLocaleChange={rememberLocale}
			/>
		{:else if id === 'notifications'}
			<WebhookList />
		{:else if id === 'bus'}
			<BusSection />
		{:else if id === 'advanced'}
			<AdvancedSection {user} onClearLocalData={() => (clearingLocal = true)} />
		{/if}
	{/snippet}
</SettingsLayout>

<ConfirmModal
	bind:open={clearingLocal}
	tone="danger"
	title={t('settings.clear.title')}
	description={t('settings.clear.description')}
	confirmLabel={t('settings.clear.confirm')}
	onConfirm={clearLocalData}
/>
