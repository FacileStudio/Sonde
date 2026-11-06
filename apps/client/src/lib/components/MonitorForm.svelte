<script lang="ts">
	import { Button, Field, Input, Select, Switch, icons } from '@facile/muse';
	import type { MonitorInput } from '$lib/backend';
	import { t } from '$lib/i18n/index.svelte';

	let {
		value,
		submitLabel,
		busy = false
	}: {
		value: MonitorInput;
		submitLabel?: string;
		busy?: boolean;
	} = $props();

	const types = $derived([
		{ value: 'http', label: t('form.types.http') },
		{ value: 'tcp', label: t('form.types.tcp') },
		{ value: 'push', label: t('form.types.push') }
	] as const);
</script>

<div class="flex flex-col gap-4">
	<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
		<Field label={t('common.name')}>
			<Input bind:value={value.name} placeholder={t('form.namePlaceholder')} required disabled={busy} />
		</Field>
		<Field label={t('common.slug')} helper={t('form.slugHelper')}>
			<Input bind:value={value.slug} placeholder="marketing-site" required disabled={busy} />
		</Field>
	</div>

	<Field label={t('common.type')}>
		<Select bind:value={value.type} disabled={busy}>
			{#each types as option (option.value)}
				<option value={option.value}>{option.label}</option>
			{/each}
		</Select>
	</Field>

	{#if value.type !== 'push'}
		<Field label={value.type === 'http' ? t('form.url') : t('form.hostPort')}>
			<Input
				bind:value={value.target}
				placeholder={value.type === 'http' ? 'https://example.com/health' : 'db.internal:5432'}
				required
				disabled={busy}
			/>
		</Field>
	{/if}

	{#if value.type === 'http'}
		<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
			<Field label={t('form.expectedStatus')}>
				<Input type="number" bind:value={value.expected_status} min="100" max="599" disabled={busy} />
			</Field>
			<Field label={t('form.expectedKeyword')} helper={t('form.keywordHelper')}>
				<Input bind:value={value.expected_keyword} placeholder="ok" disabled={busy} />
			</Field>
		</div>
	{/if}

	<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
		<Field label={t('form.interval')} helper={t('form.intervalHelper')}>
			<Input type="number" bind:value={value.interval_seconds} min="20" disabled={busy} />
		</Field>
		<Field label={t('form.timeout')} helper={t('form.timeoutHelper')}>
			<Input type="number" bind:value={value.timeout_seconds} min="1" disabled={busy} />
		</Field>
	</div>

	<Switch bind:checked={value.enabled} label={t('form.enabled')} disabled={busy} />

	<div class="flex">
		<Button type="submit" icon={icons.check} disabled={busy}>
			{busy ? t('common.saving') : (submitLabel ?? t('common.saveChanges'))}
		</Button>
	</div>
</div>
