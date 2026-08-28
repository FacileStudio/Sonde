<script lang="ts">
	import { Button, Field, Input, Select, Switch, icons } from '@facile/muse';
	import type { MonitorInput } from '$lib/backend';

	let {
		value,
		submitLabel = 'Save changes',
		busy = false
	}: {
		value: MonitorInput;
		submitLabel?: string;
		busy?: boolean;
	} = $props();

	const types = [
		{ value: 'http', label: 'HTTP(S) — probe a URL for status code and keyword' },
		{ value: 'tcp', label: 'TCP — check that a host accepts connections' },
		{ value: 'push', label: 'Push — a cron job pings Sonde, silence means down' }
	] as const;
</script>

<div class="flex flex-col gap-4">
	<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
		<Field label="Name">
			<Input bind:value={value.name} placeholder="Marketing site" required disabled={busy} />
		</Field>
		<Field label="Slug" helper="Identifies the monitor on public status pages.">
			<Input bind:value={value.slug} placeholder="marketing-site" required disabled={busy} />
		</Field>
	</div>

	<Field label="Type">
		<Select bind:value={value.type} disabled={busy}>
			{#each types as option (option.value)}
				<option value={option.value}>{option.label}</option>
			{/each}
		</Select>
	</Field>

	{#if value.type !== 'push'}
		<Field label={value.type === 'http' ? 'URL' : 'Host and port'}>
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
			<Field label="Expected status">
				<Input type="number" bind:value={value.expected_status} min="100" max="599" disabled={busy} />
			</Field>
			<Field label="Expected keyword" helper="Optional. The body must contain it.">
				<Input bind:value={value.expected_keyword} placeholder="ok" disabled={busy} />
			</Field>
		</div>
	{/if}

	<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
		<Field label="Interval (seconds)" helper="At least 20.">
			<Input type="number" bind:value={value.interval_seconds} min="20" disabled={busy} />
		</Field>
		<Field label="Timeout (seconds)" helper="Must be shorter than the interval.">
			<Input type="number" bind:value={value.timeout_seconds} min="1" disabled={busy} />
		</Field>
	</div>

	<Switch bind:checked={value.enabled} label="Checking enabled" disabled={busy} />

	<div class="flex">
		<Button type="submit" icon={icons.check} disabled={busy}>
			{busy ? 'Saving…' : submitLabel}
		</Button>
	</div>
</div>
