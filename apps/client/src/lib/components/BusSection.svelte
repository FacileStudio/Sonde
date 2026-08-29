<script lang="ts">
	/*
	 * The event bus, as an operator reads it. CHARTE §14 calls this shape a
	 * connection section: the state is not a boolean, because "not connected"
	 * hides three situations with three different fixes, and only the in-flight
	 * states pulse.
	 *
	 * There is deliberately no "gave up" state. The publisher retries its first
	 * dial forever, so an unreachable bus is always connecting or reconnecting
	 * and never a terminal state that only a restart can leave.
	 */
	import { onMount } from 'svelte';
	import {
		Alert,
		Badge,
		Button,
		SettingsRow,
		SettingsSection,
		SecretField,
		Skeleton,
		StatusDot,
		icons
	} from '@facile/muse';
	import { backend, type Integrations, type BusState } from '$lib/backend';
	import { since } from '$lib/format';

	let integrations = $state<Integrations | null>(null);
	let loading = $state(true);
	let error = $state('');

	const TONE: Record<BusState, 'success' | 'warning' | 'danger' | 'neutral'> = {
		connected: 'success',
		connecting: 'warning',
		reconnecting: 'warning',
		disabled: 'neutral'
	};

	const LABEL: Record<BusState, string> = {
		connected: 'Connected',
		connecting: 'Connecting',
		reconnecting: 'Reconnecting',
		disabled: 'Disabled'
	};

	const bus = $derived(integrations?.bus);
	const inFlight = $derived(bus?.state === 'connecting' || bus?.state === 'reconnecting');

	async function load() {
		loading = true;
		try {
			integrations = await backend.integrations();
			error = '';
		} catch (err) {
			error = err instanceof Error ? err.message : 'Could not read this instance’s integrations.';
		} finally {
			loading = false;
		}
	}

	onMount(load);
</script>

<SettingsSection
	title="Antenne"
	description="Sonde publishes every monitor transition to the Antenne bus, and Antenne decides who hears about it. Sonde does not fan out to humans itself."
>
	{#snippet actions()}
		<Button variant="outline" size="sm" icon={icons.refresh} disabled={loading} onclick={load}>
			Refresh
		</Button>
	{/snippet}

	{#if error}
		<Alert tone="danger" title="Could not read the bus state">{error}</Alert>
	{:else if loading && !bus}
		<Skeleton class="h-24 w-full" />
	{:else if bus}
		<SettingsRow
			label="Connection"
			description={bus.configured
				? 'Publishing runs in the background and retries on its own. A restart is never the fix.'
				: 'ANTENNE_INSTANCE is empty on this instance, so nothing is published. Set it, with ANTENNE_SECRET, and restart.'}
		>
			<StatusDot tone={TONE[bus.state]} label={LABEL[bus.state]} pulse={inFlight} />
		</SettingsRow>

		{#if bus.configured}
			<SettingsRow stacked>
				<SecretField label="Instance" value={bus.instance} sensitive={false} class="w-full" />
			</SettingsRow>

			<SettingsRow label="App identity" description="The name events are attributed to on the bus.">
				<span class="font-fc-mono text-fc-sm text-fc-fg">
					{bus.app}{bus.instance_id ? `:${bus.instance_id}` : ''}
				</span>
			</SettingsRow>

			{#each bus.channels as channel (channel)}
				<SettingsRow
					label={channel === 'monitor.down' ? 'Outage channel' : 'Recovery channel'}
					description={channel}
				>
					<Badge tone="neutral">emitting</Badge>
				</SettingsRow>
			{/each}

			<SettingsRow
				label="Events published"
				description="Since this container started. A restart resets the count, not the delivery."
			>
				<span class="font-fc-mono text-fc-sm text-fc-fg">{bus.published}</span>
			</SettingsRow>

			{#if bus.dropped > 0}
				<SettingsRow
					label="Events dropped"
					description="Emitted while the bus was unreachable. Sonde does not buffer: a monitor that keeps probing matters more than a queue that keeps growing."
				>
					<span class="font-fc-mono text-fc-sm text-fc-danger">{bus.dropped}</span>
				</SettingsRow>
			{/if}

			<SettingsRow label="Last event" description="The most recent transition that left this box.">
				<span class="text-fc-sm text-fc-fg-muted">{since(bus.last_event_at)}</span>
			</SettingsRow>

			{#if bus.last_error}
				<SettingsRow stacked>
					<Alert tone="warning" title="Last error from the bus">
						<span class="font-fc-mono text-fc-xs">{bus.last_error}</span>
					</Alert>
				</SettingsRow>
			{/if}
		{/if}
	{/if}
</SettingsSection>

{#if integrations}
	<SettingsSection
		title="History"
		description="How much of the past this instance can still answer questions about."
	>
		<SettingsRow
			label="Raw checks"
			description="Every individual probe. Set with SONDE_RAW_RETENTION_DAYS."
		>
			<span class="font-fc-mono text-fc-sm text-fc-fg">{integrations.retention.raw_days} days</span>
		</SettingsRow>
		<SettingsRow
			label="Hourly rollups"
			description="What every uptime percentage and chart is read from, so this is the real limit on how far back a window can go."
		>
			<span class="font-fc-mono text-fc-sm text-fc-fg">
				{integrations.retention.hourly_months} months
			</span>
		</SettingsRow>
		<SettingsRow
			label="Failure threshold"
			description="Consecutive failed checks before an incident opens and the bus hears about it."
		>
			<span class="font-fc-mono text-fc-sm text-fc-fg">
				{integrations.retention.failure_threshold}
			</span>
		</SettingsRow>
		<SettingsRow
			label="Log shipping"
			description={integrations.journal.configured
				? integrations.journal.url
				: 'JOURNAL_URL is empty, so logs stay in this container.'}
		>
			<StatusDot
				tone={integrations.journal.configured ? 'success' : 'neutral'}
				label={integrations.journal.configured ? 'Journal' : 'Local only'}
			/>
		</SettingsRow>
	</SettingsSection>
{/if}
