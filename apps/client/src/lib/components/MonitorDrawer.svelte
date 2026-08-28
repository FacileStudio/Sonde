<script lang="ts">
	import { Alert, Button, Drawer, Field, Input, Select, Spinner, icons } from '@facile/muse';
	import { backend, type Monitor, type MonitorInput } from '$lib/backend';

	let {
		open = $bindable(false),
		onCreated
	}: {
		open?: boolean;
		onCreated: (monitor: Monitor) => void;
	} = $props();

	const types = [
		{ value: 'http', label: 'HTTP(S) — probe a URL' },
		{ value: 'tcp', label: 'TCP — check a port accepts connections' },
		{ value: 'push', label: 'Push — a cron job pings Sonde' }
	] as const;

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
			error = err instanceof Error ? err.message : 'Could not create the monitor.';
		} finally {
			submitting = false;
		}
	}
</script>

<Drawer
	bind:open
	title="New monitor"
	description="Checks start on the next tick, every 60 seconds. Tune the rest on the monitor itself."
	showClose
>
	<form id="new-monitor" class="flex flex-col gap-4" onsubmit={create}>
		<Field label="Name" helper="How it appears in your monitor list.">
			<Input bind:value={name} placeholder="Marketing site" required disabled={submitting} />
		</Field>

		<Field label="Slug" helper="Identifies the monitor on public status pages.">
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

		<Field label="Type">
			<Select bind:value={type} disabled={submitting}>
				{#each types as option (option.value)}
					<option value={option.value}>{option.label}</option>
				{/each}
			</Select>
		</Field>

		{#if needsTarget}
			<Field
				label={type === 'http' ? 'URL' : 'Host and port'}
				helper={type === 'http'
					? 'A 200 with any body counts as up until you narrow it.'
					: 'Sonde only opens the connection, it sends nothing.'}
			>
				<Input
					bind:value={target}
					placeholder={type === 'http' ? 'https://example.com/health' : 'db.internal:5432'}
					required
					disabled={submitting}
				/>
			</Field>
		{:else}
			<Alert tone="info" title="Push monitor">
				Sonde issues a heartbeat URL once the monitor exists. Silence past the interval counts as
				down.
			</Alert>
		{/if}

		{#if error}
			<Alert tone="danger" title="Not added">{error}</Alert>
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
				Cancel
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
				{submitting ? 'Adding…' : 'Add monitor'}
			</Button>
		</div>
	{/snippet}
</Drawer>
