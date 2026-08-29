<script lang="ts">
	/*
	 * The Advanced tab: what this instance is, and the danger zone at the
	 * bottom of it. Danger never gets a tab of its own (CHARTE §14) — a
	 * destructive action should cost a scroll, not sit one mis-click from
	 * every other tab all day.
	 */
	import { page } from '$app/state';
	import { Button, SecretField, SettingsRow, SettingsSection, icons, toast } from '@facile/muse';
	import { backend, type AuthUser } from '$lib/backend';

	let {
		user,
		ssoOnly,
		onClearLocalData
	}: {
		user: AuthUser | null;
		ssoOnly: boolean;
		onClearLocalData: () => void;
	} = $props();

	const memberSince = $derived(
		user?.created_at ? new Date(user.created_at).toLocaleDateString() : '\u2014'
	);

	async function exportConfig() {
		try {
			await backend.exportConfig();
		} catch (err) {
			toast.danger(err instanceof Error && err.message ? err.message : 'Export failed.');
		}
	}
</script>

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
			<Button variant="danger" icon={icons.remove} onclick={onClearLocalData}>
				Clear local data
			</Button>
		</SettingsRow>
	</SettingsSection>
