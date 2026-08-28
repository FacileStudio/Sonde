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
		ProfileCard,
		SecretField,
		SettingsRow,
		SettingsSection,
		Tabs,
		icons,
		toast
	} from '@facile/muse';
	import { backend, type AuthUser } from '$lib/backend';
	import { setTheme, theme, type ThemeMode } from '$lib/theme.svelte';

	const app = getContext<{ user: AuthUser | null; setUser: (u: AuthUser) => void }>('app');

	const SECTIONS = [
		{ id: 'profile', label: 'Profile', icon: icons.userCircle },
		{ id: 'appearance', label: 'Appearance', icon: icons.palette },
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
		profileSaving = true;
		try {
			/* The PATCH answers with the user minus created_at, so the fresh profile comes from
			   /auth/me — the sidebar card and the meta rows read the same object. */
			await backend.updateProfile(
				ssoOnly ? { name: profileName } : { name: profileName, email: profileEmail }
			);
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

<div class="flex flex-col gap-1">
	<h1 class="text-fc-2xl font-semibold text-fc-fg">Settings</h1>
	<p class="text-fc-sm text-fc-fg-muted">Your account, this browser, and this instance.</p>
</div>

<div class="flex flex-col gap-4">
	<Tabs items={tabs} value={section} label="Settings sections" />
	<Divider class="my-0" />
</div>

<div class="flex flex-col gap-10">
	{#if section === 'profile'}
		<ProfileCard
			name={user?.name?.trim() || user?.email || ''}
			email={user?.email ?? ''}
			role={user?.is_admin ? 'admin' : 'member'}
			meta={[
				{ label: 'Member since', value: memberSince },
				{ label: 'Role', value: user?.is_admin ? 'Administrator' : 'Member' }
			]}
		/>

		<SettingsSection title="Identity" description="How Sonde names you across the app.">
			<SettingsRow label="Display name" stacked>
				<Field>
					<Input bind:value={profileName} placeholder="Your name" />
				</Field>
			</SettingsRow>
			<SettingsRow
				label="Email"
				description={ssoOnly
					? 'Owned by the identity provider at sso.facile.studio. Change it there and it lands here at your next sign-in.'
					: 'Also the address you sign in with.'}
				stacked
			>
				<Field>
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
	{:else}
		<SettingsSection
			title="Configuration export"
			description="Every monitor and status page as JSON, with no secrets in it. Keep it in git."
		>
			<SettingsRow
				label="Download sonde-config.json"
				description={user?.is_admin
					? 'A snapshot of what this instance is watching, taken now.'
					: 'Administrators only. Ask whoever runs this instance.'}
			>
				<Button
					variant="outline"
					icon={icons.download}
					disabled={!user?.is_admin}
					onclick={exportConfig}
				>
					Export
				</Button>
			</SettingsRow>
		</SettingsSection>

		<SettingsSection title="Instance" description="What this browser is talking to.">
			<SettingsRow stacked>
				<SecretField
					label="Instance URL"
					value={page.url.origin}
					sensitive={false}
					class="w-full"
				/>
			</SettingsRow>
			<SettingsRow label="API base path" description="Every request is scoped under it.">
				<span class="font-fc-mono text-fc-sm text-fc-fg">{backend.baseUrl}</span>
			</SettingsRow>
			<SettingsRow label="API reference" description="Every endpoint this instance serves.">
				<Button variant="outline" href="/docs" iconRight={icons.arrow}>Open</Button>
			</SettingsRow>
		</SettingsSection>

		<SettingsSection title="Account" description="Facts an operator may ask you for.">
			<SettingsRow stacked>
				<SecretField
					label="User ID"
					value={String(user?.id ?? '')}
					sensitive={false}
					class="w-full"
				/>
			</SettingsRow>
			<SettingsRow label="Member since" description="When this account was created.">
				<span class="text-fc-sm text-fc-fg-muted">{memberSince}</span>
			</SettingsRow>
			<SettingsRow label="Single sign-on" description="Whether local passwords are disabled.">
				<span class="font-fc-mono text-fc-sm text-fc-fg">{ssoOnly ? 'required' : 'optional'}</span>
			</SettingsRow>
		</SettingsSection>

		<SettingsSection
			title="Danger zone"
			description="Irreversible from this browser. Your monitors, incidents and status pages are untouched."
		>
			<SettingsRow
				label="Clear local data"
				description="Removes the theme preference stored here, then signs you out. Sonde has no self-service account deletion, an administrator has to remove the account server-side."
			>
				<Button variant="danger" icon={icons.remove} onclick={() => (clearingLocal = true)}>
					Clear local data
				</Button>
			</SettingsRow>
		</SettingsSection>
	{/if}
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
