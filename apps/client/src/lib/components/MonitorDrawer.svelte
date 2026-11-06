<script lang="ts">
	import { Alert, Button, Drawer, Field, Input, Select, Spinner, icons, toast } from '@facile/muse';
	import { backend, type Monitor, type MonitorInput } from '$lib/backend';
	import { t } from '$lib/i18n/index.svelte';

	let {
		open = $bindable(false),
		onCreated
	}: {
		open?: boolean;
		onCreated: (monitor: Monitor) => void;
	} = $props();

	const types = $derived([
		{ value: 'http', label: t('drawer.types.http') },
		{ value: 'tcp', label: t('drawer.types.tcp') },
		{ value: 'push', label: t('drawer.types.push') }
	] as const);

	let name = $state('');
	let slug = $state('');
	let slugEdited = $state(false);
	let type = $state<Monitor['type']>('http');
	let target = $state('');
	let error = $state('');
	let submitting = $state(false);

	const slugify = (value: string) =>
		value
			.toLowerCase()
			.replace(/[^a-z0-9]+/g, '-')
			.replace(/^-+|-+$/g, '')
			.slice(0, 60);

	const effectiveSlug = $derived(slugEdited ? slug : slugify(name));
	const needsTarget = $derived(type !== 'push');
	const ready = $derived(
		Boolean(name.trim()) && Boolean(effectiveSlug) && (!needsTarget || Boolean(target.trim()))
	);

	/* Reopening must never show the previous attempt's error or half-typed target. */
	$effect(() => {
		if (!open) {
			name = '';
			slug = '';
			slugEdited = false;
			type = 'http';
			target = '';
			error = '';
		}
	});

	async function create(event: SubmitEvent) {
		event.preventDefault();
		if (submitting) return;
		error = '';
		submitting = true;
		const input: MonitorInput = {
			slug: effectiveSlug,
			name: name.trim(),
			type,
			target: needsTarget ? target.trim() : '',
			interval_seconds: 60,
			timeout_seconds: 10,
			expected_status: type === 'http' ? 200 : 0,
			expected_keyword: '',
			enabled: true
		};
		try {
			const monitor = await backend.createMonitor(input);
			open = false;
			onCreated(monitor);
		} catch (err) {
			error = err instanceof Error ? err.message : t('drawer.createFailed');
			/* The Alert lives inside the sheet. A toast survives it being closed. */
			toast.danger(error);
		} finally {
			submitting = false;
		}
	}
</script>

<Drawer
	bind:open
	title={t('monitors.new')}
	description={t('drawer.description')}
	showClose
	dismissible={!submitting}
>
	<form id="new-monitor" class="flex flex-col gap-4" onsubmit={create}>
		<Field label={t('common.name')} helper={t('drawer.nameHelper')}>
			<Input bind:value={name} placeholder={t('form.namePlaceholder')} required disabled={submitting} />
		</Field>

		<Field label={t('common.slug')} helper={t('form.slugHelper')}>
			<Input
				value={effectiveSlug}
				oninput={(event: Event) => {
					slugEdited = true;
					slug = (event.currentTarget as HTMLInputElement).value;
				}}
				placeholder="marketing-site"
				required
				disabled={submitting}
			/>
		</Field>

		<Field label={t('common.type')}>
			<Select bind:value={type} disabled={submitting}>
				{#each types as option (option.value)}
					<option value={option.value}>{option.label}</option>
				{/each}
			</Select>
		</Field>

		{#if needsTarget}
			<Field
				label={type === 'http' ? t('form.url') : t('form.hostPort')}
				helper={type === 'http' ? t('drawer.httpHelper') : t('drawer.tcpHelper')}
			>
				<Input
					bind:value={target}
					placeholder={type === 'http' ? 'https://example.com/health' : 'db.internal:5432'}
					required
					disabled={submitting}
				/>
			</Field>
		{:else}
			<Alert tone="info" title={t('drawer.push.title')}>{t('drawer.push.text')}</Alert>
		{/if}

		{#if error}
			<Alert tone="danger" title={t('drawer.notAdded')}>{error}</Alert>
		{/if}
	</form>

	{#snippet footer()}
		<div class="flex gap-2">
			<Button
				variant="ghost"
				size="lg"
				class="flex-1"
				disabled={submitting}
				onclick={() => (open = false)}
			>
				{t('common.cancel')}
			</Button>
			<Button
				type="submit"
				form="new-monitor"
				size="lg"
				icon={icons.plus}
				class="flex-1"
				disabled={submitting || !ready}
			>
				{#if submitting}<Spinner size="sm" />{/if}
				{submitting ? t('drawer.adding') : t('drawer.add')}
			</Button>
		</div>
	{/snippet}
</Drawer>
