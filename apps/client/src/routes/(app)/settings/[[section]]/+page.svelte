<script lang="ts">
	import { getContext, onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import {
		Button,
		ConfirmModal,
		Divider,
		Field,
		Input,
		OptionCards,
		PageHeader,
		ProfileCard,
		SecretField,
		SettingsRow,
		SettingsSection,
		Tabs,
		icons,
		toast
	} from '@facile/muse';
	import { backend, type AuthUser, type ProfileUpdate } from '$lib/backend';
	import { date } from '$lib/format';
	import { setSessionLocale, t } from '$lib/i18n/index.svelte';
	import { setTheme, theme, type ThemeMode } from '$lib/theme.svelte';
	import WebhookList from '$lib/components/WebhookList.svelte';
	import BusSection from '$lib/components/BusSection.svelte';
	import AdvancedSection from '$lib/components/AdvancedSection.svelte';

	const app = getContext<{ user: AuthUser | null; setUser: (u: AuthUser) => void }>('app');

	/*
	 * Order is CHARTE §14's: identity, then this browser, then the two outbound
	 * paths, then the instance. Bus sits next to Notifications because they are
	 * the same question asked twice — Antenne is where alerts are meant to go,
	 * and a webhook is what you reach for when they are not.
	 */
	const SECTIONS = [
		{ id: 'profile', icon: icons.userCircle },
		{ id: 'appearance', icon: icons.palette },
		{ id: 'bus', icon: icons.plug },
		{ id: 'notifications', icon: icons.notification },
		{ id: 'advanced', icon: icons.server }
	] as const;

	/*
	 * The section is a real route, so a link to /settings/advanced opens on Advanced, a reload
	 * keeps you there and browser-back walks the sections — none of which local `$state` tabs
	 * give you (CHARTE §14). An unknown slug falls back to Profile rather than rendering nothing.
	 */
	const section = $derived(
		SECTIONS.some((s) => s.id === page.params.section) ? page.params.section! : 'profile'
	);
	const tabs = $derived(
		SECTIONS.map((s) => ({ ...s, label: t(`settings.section.${s.id}`), href: `/settings/${s.id}` }))
	);

	let profileName = $state('');
	let profileEmail = $state('');
	let profileSaving = $state(false);
	let loggingOut = $state(false);
	let clearingLocal = $state(false);
	let themeMode = $state<string>(theme.mode);

	const user = $derived(app.user);
	const memberSince = $derived(
		user?.created_at ? date(user.created_at) : '—'
	);

	onMount(() => {
		profileName = user?.name ?? '';
		profileEmail = user?.email ?? '';
	});

	$effect(() => {
		setTheme(themeMode as ThemeMode);
	});

	async function saveProfile() {
		/* Both fields are optional on the API, and a present-but-empty `name` is rejected as
		   "invalid name". An SSO account that arrived with no name claim starts blank, so
		   always sending the field made its email unsaveable too. Omit it instead. */
		const update: ProfileUpdate = {};
		const trimmedName = profileName.trim();
		if (trimmedName) update.name = trimmedName;
		if (!update.name) {
			toast.danger(t('settings.nameRequired'));
			return;
		}

		profileSaving = true;
		try {
			/* The PATCH answers with the user minus created_at, so the fresh profile comes from
			   /auth/me — the sidebar card and the meta rows read the same object. */
			await backend.updateProfile(update);
			const fresh = await backend.me();
			app.setUser(fresh.user);
			profileName = fresh.user.name ?? '';
			profileEmail = fresh.user.email;
			setSessionLocale(fresh.user.locale);
			toast.success(t('settings.profileUpdated'));
		} catch (err) {
			toast.danger(message(err, t('settings.profileFailed')));
		}
		profileSaving = false;
	}

	/*
	 * The cookie is the half that matters, and the redirect happens either way: a logout that
	 * fails server-side must still get them off the app.
	 */
	async function logout() {
		await backend.logout().catch(() => undefined);
		goto('/login');
	}

	function clearLocalData() {
		localStorage.removeItem('sonde-theme');
		logout();
	}

	function message(error: unknown, fallback: string): string {
		return error instanceof Error && error.message ? error.message : fallback;
	}

	const themeOptions = $derived([
		{ value: 'system', label: t('settings.theme.system'), icon: icons.monitor },
		{ value: 'light', label: t('settings.theme.light'), icon: icons.sun },
		{ value: 'dark', label: t('settings.theme.dark'), icon: icons.moon }
	]);
</script>

<svelte:head>
	<title>{t('nav.settings')} · Sonde</title>
</svelte:head>

<div class="flex flex-col gap-8">
	<div class="flex flex-col gap-6">
		<PageHeader title={t('nav.settings')} description={t('settings.description')} />
		<div class="flex flex-col gap-4">
			<Tabs items={tabs} value={section} label={t('settings.sections')} />
			<Divider class="my-0" />
		</div>
	</div>

	<div class="flex flex-col gap-10">
		{#if section === 'profile'}
		<ProfileCard
			name={user?.name?.trim() || user?.email || ''}
			email={user?.email ?? ''}
			avatar={user?.avatar_url || undefined}
			role={user?.is_admin ? t('settings.role.admin') : t('settings.role.member')}
			meta={[
				{ label: t('settings.memberSince'), value: memberSince },
				{ label: t('settings.role'), value: user?.is_admin ? t('settings.role.admin') : t('settings.role.member') }
			]}
		/>

		<SettingsSection title={t('settings.identity.title')} description={t('settings.identity.description')}>
			<!-- The row's label is the input's `<label for>`: SettingsRow renders a plain <span>
			     without `for`, and a bare Field emits no label at all, so the control had no
			     accessible name. Field takes the same id so it stops minting its own. -->
			<SettingsRow label={t('settings.displayName')} for="profile-name" stacked>
				<Field for="profile-name">
					<Input bind:value={profileName} placeholder={t('settings.namePlaceholder')} />
				</Field>
			</SettingsRow>
			<SettingsRow
				label={t('settings.email')}
				for="profile-email"
				description={t('settings.emailOwned')}
				stacked
			>
				<Field for="profile-email">
					<Input bind:value={profileEmail} type="email" disabled />
				</Field>
			</SettingsRow>
			<SettingsRow>
				<Button icon={icons.check} disabled={profileSaving} onclick={saveProfile}>
					{profileSaving ? t('common.saving') : t('settings.saveProfile')}
				</Button>
			</SettingsRow>
		</SettingsSection>

		<SettingsSection
			title={t('settings.session.title')}
			description={t('settings.session.description')}
		>
			<SettingsRow label={t('settings.logout')} description={t('settings.logout.description')}>
				<Button variant="outline" icon={icons.logout} onclick={() => (loggingOut = true)}>
					{t('settings.logout')}
				</Button>
			</SettingsRow>
		</SettingsSection>
	{:else if section === 'appearance'}
		<SettingsSection title={t('settings.theme.title')} description={t('settings.theme.description')}>
			<SettingsRow
				label={t('settings.theme.scheme')}
				description={t('settings.theme.schemeHint')}
				stacked
			>
				<OptionCards
					options={themeOptions}
					bind:value={themeMode}
					name="theme-mode"
					label={t('settings.theme.scheme')}
				/>
			</SettingsRow>
		</SettingsSection>
	{:else if section === 'bus'}
		<BusSection />
	{:else if section === 'notifications'}
		<WebhookList />
	{:else}
		<AdvancedSection {user} onClearLocalData={() => (clearingLocal = true)} />
	{/if}
	</div>
</div>

<ConfirmModal
	bind:open={loggingOut}
	title={t('settings.logout.title')}
	description={t('settings.logout.confirmText')}
	confirmLabel={t('settings.logout')}
	onConfirm={logout}
/>

<ConfirmModal
	bind:open={clearingLocal}
	tone="danger"
	title={t('settings.clear.title')}
	description={t('settings.clear.description')}
	confirmLabel={t('settings.clear.confirm')}
	onConfirm={clearLocalData}
/>
