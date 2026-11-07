<script lang="ts">
	import { page } from '$app/state';
	import { Button, DangerZone, SecretField, SettingsRow, SettingsSection, icons, toast } from '@facile/muse';
	import { backend, type AuthUser } from '$lib/backend';
	import { date } from '$lib/format';
	import { t } from '$lib/i18n/index.svelte';

	let {
		user,
		onClearLocalData
	}: {
		user: AuthUser | null;
		onClearLocalData: () => void;
	} = $props();

	const memberSince = $derived(
		user?.created_at ? date(user.created_at) : '\u2014'
	);

	async function exportConfig() {
		try {
			await backend.exportConfig();
		} catch (err) {
			toast.danger(err instanceof Error && err.message ? err.message : t('advanced.export.failed'));
		}
	}
</script>

	<SettingsSection
		title={t('advanced.export.title')}
		description={t('advanced.export.description')}
	>
		<SettingsRow
			label={t('advanced.export.label')}
			description={user?.is_admin ? t('advanced.export.adminHint') : t('advanced.export.memberHint')}
		>
			<Button
				variant="outline"
				icon={icons.download}
				disabled={!user?.is_admin}
				onclick={exportConfig}
			>
				{t('advanced.export.button')}
			</Button>
		</SettingsRow>
	</SettingsSection>

	<SettingsSection title={t('advanced.instance.title')} description={t('advanced.instance.description')}>
		<SettingsRow stacked>
			<SecretField
				label={t('advanced.instance.url')}
				value={page.url.origin}
				sensitive={false}
				class="w-full"
			/>
		</SettingsRow>
		<SettingsRow label={t('advanced.api.base')} description={t('advanced.api.baseHint')}>
			<span class="font-fc-mono text-fc-sm text-fc-fg">{backend.baseUrl}</span>
		</SettingsRow>
		<SettingsRow label={t('advanced.api.reference')} description={t('advanced.api.referenceHint')}>
			<Button variant="outline" href="/docs" iconRight={icons.arrow}>{t('common.open')}</Button>
		</SettingsRow>
	</SettingsSection>

	<SettingsSection title={t('advanced.account.title')} description={t('advanced.account.description')}>
		<SettingsRow stacked>
			<SecretField
				label={t('advanced.account.userId')}
				value={String(user?.id ?? '')}
				sensitive={false}
				class="w-full"
			/>
		</SettingsRow>
		<SettingsRow label={t('settings.memberSince')} description={t('advanced.account.createdHint')}>
			<span class="text-fc-sm text-fc-fg-muted">{memberSince}</span>
		</SettingsRow>
	</SettingsSection>

	<DangerZone title={t('advanced.danger.title')} description={t('advanced.danger.description')}>
		<Button variant="danger" icon={icons.remove} onclick={onClearLocalData}>
			{t('advanced.danger.clear')}
		</Button>
	</DangerZone>
