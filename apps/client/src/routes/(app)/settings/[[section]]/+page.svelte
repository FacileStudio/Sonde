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
		{ id: 'profile', label: 'Profile', icon: icons.userCircle },
		{ id: 'appearance', label: 'Appearance', icon: icons.palette },
		{ id: 'bus', label: 'Bus', icon: icons.plug },
		{ id: 'notifications', label: 'Notifications', icon: icons.notification },
		{ id: 'advanced', label: 'Advanced', icon: icons.server }
	] as const;

	/*
	 * The section is a real route, so a link to /settings/advanced opens on Advanced, a reload
	 * keeps you there and browser-back walks the sections — none of which local `$state` tabs
	 * give you (CHARTE §14). An unknown slug falls back to Profile rather than rendering nothing.
	 */
	const section = $derived(
		SECTIONS.some((s) => s.id === page.params.section) ? page.params.section! : 'profile'
	);
	const tabs = SECTIONS.map((s) => ({ ...s, href: `/settings/${s.id}` }));

	let profileName = $state('');
	let profileEmail = $state('');
	let profileSaving = $state(false);
	let ssoOnly = $state(false);
	let loggingOut = $state(false);
	let clearingLocal = $state(false);
	let themeMode = $state<string>(theme.mode);

	const user = $derived(app.user);
	const memberSince = $derived(
		user?.created_at ? new Date(user.created_at).toLocaleDateString() : '—'
	);

	onMount(async () => {
		profileName = user?.name ?? '';
		profileEmail = user?.email ?? '';
		try {
			ssoOnly = (await backend.authConfig()).sso_only;
		} catch {
			toast.danger('Could not read this instance’s auth configuration.');
		}
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
		if (!ssoOnly) update.email = profileEmail.trim();
		if (!update.name && !update.email) {
			toast.danger('Enter a display name to save.');
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
			toast.success('Profile updated.');
		} catch (err) {
			toast.danger(message(err, 'Could not update your profile.'));
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

	const themeOptions = [
		{ value: 'system', label: 'System', icon: icons.monitor },
		{ value: 'light', label: 'Light', icon: icons.sun },
		{ value: 'dark', label: 'Dark', icon: icons.moon }
	];

	async function exportConfig() {
		try {
			await backend.exportConfig();
		} catch (err) {
			toast.danger(message(err, 'Export failed.'));
		}
	}
</script>

<svelte:head>
	<title>Settings — Sonde</title>
</svelte:head>

<div class="flex flex-col gap-8">
	<div class="flex flex-col gap-6">
		<PageHeader title="Settings" description="Your account, this browser, and this instance." />
		<div class="flex flex-col gap-4">
			<Tabs items={tabs} value={section} label="Settings sections" />
			<Divider class="my-0" />
		</div>
	</div>

	<div class="flex flex-col gap-10">
		{#if section === 'profile'}
		<ProfileCard
			name={user?.name?.trim() || user?.email || ''}
			email={user?.email ?? ''}
			avatar={user?.avatar_url || undefined}
			role={user?.is_admin ? 'admin' : 'member'}
			meta={[
				{ label: 'Member since', value: memberSince },
				{ label: 'Role', value: user?.is_admin ? 'Administrator' : 'Member' }
			]}
		/>

		<SettingsSection title="Identity" description="How Sonde names you across the app.">
			<!-- The row's label is the input's `<label for>`: SettingsRow renders a plain <span>
			     without `for`, and a bare Field emits no label at all, so the control had no
			     accessible name. Field takes the same id so it stops minting its own. -->
			<SettingsRow label="Display name" for="profile-name" stacked>
				<Field for="profile-name">
					<Input bind:value={profileName} placeholder="Your name" />
				</Field>
			</SettingsRow>
			<SettingsRow
				label="Email"
				for="profile-email"
				description={ssoOnly
					? 'Owned by the identity provider at sso.facile.studio. Change it there and it lands here at your next sign-in.'
					: 'Also the address you sign in with.'}
				stacked
			>
				<Field for="profile-email">
					<Input bind:value={profileEmail} type="email" disabled={ssoOnly} />
				</Field>
			</SettingsRow>
			<SettingsRow>
				<Button icon={icons.check} disabled={profileSaving} onclick={saveProfile}>
					{profileSaving ? 'Saving…' : 'Save profile'}
				</Button>
			</SettingsRow>
		</SettingsSection>

		<SettingsSection
			title="Session"
			description="This browser only. Your monitors keep running either way."
		>
			<SettingsRow
				label="Log out"
				description="You will need to sign in again to reach the dashboard."
			>
				<Button variant="outline" icon={icons.logout} onclick={() => (loggingOut = true)}>
					Log out
				</Button>
			</SettingsRow>
		</SettingsSection>
	{:else if section === 'appearance'}
		<SettingsSection
			title="Theme"
			description="Applied to this browser. Every device you sign in from keeps its own choice."
		>
			<SettingsRow
				label="Colour scheme"
				description="System follows your OS for as long as the app is open."
				stacked
			>
				<OptionCards
					options={themeOptions}
					bind:value={themeMode}
					name="theme-mode"
					label="Colour scheme"
				/>
			</SettingsRow>
		</SettingsSection>
	{:else if section === 'bus'}
		<BusSection />
	{:else if section === 'notifications'}
		<WebhookList />
	{:else}
		<AdvancedSection {user} {ssoOnly} onClearLocalData={() => (clearingLocal = true)} />
	{/if}
	</div>
</div>

<ConfirmModal
	bind:open={loggingOut}
	title="Log out of Sonde?"
	description="Your monitors keep running. You will need to sign in again on this browser."
	confirmLabel="Log out"
	onConfirm={logout}
/>

<ConfirmModal
	bind:open={clearingLocal}
	tone="danger"
	title="Clear local data?"
	description="The theme preference stored in this browser is deleted and you are signed out. There is no undo. Your monitors, incidents and status pages stay on the server."
	confirmLabel="Clear and sign out"
	onConfirm={clearLocalData}
/>
