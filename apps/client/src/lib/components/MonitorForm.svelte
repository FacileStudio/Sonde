<script lang="ts">
	import { untrack } from 'svelte';
	import { Button, Field, Input, Select } from '@facile/muse';
	import type { MonitorInput } from '$lib/backend';

	let { value, submitLabel = 'Save', busy = false, error = '' }: {
		value: MonitorInput;
		submitLabel?: string;
		busy?: boolean;
		error?: string;
	} = $props();

	const types = [
		{ value: 'http', label: 'HTTP(S)', hint: 'Probe a URL for status code and keyword' },
		{ value: 'tcp', label: 'TCP port', hint: 'Check that a host accepts connections' },
		{ value: 'push', label: 'Push', hint: 'A cron job pings Sonde; silence means down' }
	] as const;

	let selectedType = $state<string>(untrack(() => value.type));

	function syncType() {
		value.type = selectedType as MonitorInput['type'];
	}
</script>

<div class="flex flex-col gap-6">
	{#if error}
		<div class="rounded-md border border-fc-border bg-fc-danger/10 p-3 text-fc-sm text-fc-danger">
			{error}
		</div>
	{/if}

	<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
		<Field label="Name">
			{#snippet children({ id })}
				<Input {id} bind:value={value.name} placeholder="Marketing site" required />
			{/snippet}
		</Field>
		<Field label="Slug">
			{#snippet children({ id })}
				<Input {id} bind:value={value.slug} placeholder="marketing-site" required />
			{/snippet}
		</Field>
	</div>

	<Field label="Type">
		{#snippet children({ id })}
			<Select {id} bind:value={selectedType} onchange={syncType}>
				{#each types as t (t.value)}
					<option value={t.value}>{t.label} — {t.hint}</option>
				{/each}
			</Select>
		{/snippet}
	</Field>

	{#if selectedType !== 'push'}
		<Field label={selectedType === 'http' ? 'URL' : 'Host and port'}>
			{#snippet children({ id })}
				<Input
					{id}
					bind:value={value.target}
					placeholder={selectedType === 'http' ? 'https://example.com/health' : 'db.internal:5432'}
					required
				/>
			{/snippet}
		</Field>
	{/if}

	{#if selectedType === 'http'}
		<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
			<Field label="Expected status">
				{#snippet children({ id })}
					<Input {id} type="number" bind:value={value.expected_status} min="100" max="599" />
				{/snippet}
			</Field>
			<Field label="Expected keyword (optional)">
				{#snippet children({ id })}
					<Input {id} bind:value={value.expected_keyword} placeholder="ok" />
				{/snippet}
			</Field>
		</div>
	{/if}

	<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
		<Field label="Interval (seconds)">
			{#snippet children({ id })}
				<Input {id} type="number" bind:value={value.interval_seconds} min="20" />
			{/snippet}
		</Field>
		<Field label="Timeout (seconds)">
			{#snippet children({ id })}
				<Input {id} type="number" bind:value={value.timeout_seconds} min="1" />
			{/snippet}
		</Field>
	</div>

	<Button type="submit" disabled={busy}>{busy ? 'Saving…' : submitLabel}</Button>
</div>
