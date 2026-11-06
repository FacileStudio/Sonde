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
	import { t } from '$lib/i18n/index.svelte';

	let integrations = $state<Integrations | null>(null);
	let loading = $state(true);
	let error = $state('');

	const TONE: Record<BusState, 'success' | 'warning' | 'danger' | 'neutral'> = {
		connected: 'success',
		connecting: 'warning',
		reconnecting: 'warning',
		disabled: 'neutral'
	};

	const bus = $derived(integrations?.bus);
	const inFlight = $derived(bus?.state === 'connecting' || bus?.state === 'reconnecting');

	async function load() {
		loading = true;
		try {
			integrations = await backend.integrations();
			error = '';
		} catch (err) {
			error = err instanceof Error ? err.message : t('bus.loadFailed');
		} finally {
			loading = false;
		}
	}

	onMount(load);
</script>

<SettingsSection
	title={t('bus.title')}
	description={t('bus.description')}
>
	{#snippet actions()}
		<Button variant="outline" size="sm" icon={icons.refresh} disabled={loading} onclick={load}>
			{t('common.refresh')}
		</Button>
	{/snippet}

	{#if error}
		<Alert tone="danger" title={t('bus.loadError')}>{error}</Alert>
	{:else if loading && !bus}
		<Skeleton class="h-24 w-full" />
	{:else if bus}
		<SettingsRow
			label={t('bus.connection')}
			description={bus.configured ? t('bus.connection.configured') : t('bus.connection.unconfigured')}
		>
			<StatusDot tone={TONE[bus.state]} label={t(`bus.state.${bus.state}`)} pulse={inFlight} />
		</SettingsRow>

		{#if bus.configured}
			<SettingsRow stacked>
				<SecretField label={t('bus.instance')} value={bus.instance} sensitive={false} class="w-full" />
			</SettingsRow>

			<SettingsRow label={t('bus.appIdentity')} description={t('bus.appIdentityHint')}>
				<span class="font-fc-mono text-fc-sm text-fc-fg">
					{bus.app}{bus.instance_id ? `:${bus.instance_id}` : ''}
				</span>
			</SettingsRow>

			{#each bus.channels as channel (channel)}
				<SettingsRow
					label={channel === 'monitor.down' ? t('bus.channel.down') : t('bus.channel.up')}
					description={channel}
				>
					<Badge tone="neutral">{t('bus.emitting')}</Badge>
				</SettingsRow>
			{/each}

			<SettingsRow
				label={t('bus.published')}
				description={t('bus.publishedHint')}
			>
				<span class="font-fc-mono text-fc-sm text-fc-fg">{bus.published}</span>
			</SettingsRow>

			{#if bus.dropped > 0}
				<SettingsRow
					label={t('bus.dropped')}
					description={t('bus.droppedHint')}
				>
					<span class="font-fc-mono text-fc-sm text-fc-danger">{bus.dropped}</span>
				</SettingsRow>
			{/if}

			<SettingsRow label={t('bus.lastEvent')} description={t('bus.lastEventHint')}>
				<span class="text-fc-sm text-fc-fg-muted">{since(bus.last_event_at)}</span>
			</SettingsRow>

			{#if bus.last_error}
				<SettingsRow stacked>
					<Alert tone="warning" title={t('bus.lastError')}>
						<span class="font-fc-mono text-fc-xs">{bus.last_error}</span>
					</Alert>
				</SettingsRow>
			{/if}
		{/if}
	{/if}
</SettingsSection>

{#if integrations}
	<SettingsSection
		title={t('retention.title')}
		description={t('retention.description')}
	>
		<SettingsRow
			label={t('retention.raw')}
			description={t('retention.rawHint')}
		>
			<span class="font-fc-mono text-fc-sm text-fc-fg">{t('retention.days', { count: integrations.retention.raw_days })}</span>
		</SettingsRow>
		<SettingsRow
			label={t('retention.hourly')}
			description={t('retention.hourlyHint')}
		>
			<span class="font-fc-mono text-fc-sm text-fc-fg">
				{t('retention.months', { count: integrations.retention.hourly_months })}
			</span>
		</SettingsRow>
		<SettingsRow
			label={t('retention.threshold')}
			description={t('retention.thresholdHint')}
		>
			<span class="font-fc-mono text-fc-sm text-fc-fg">
				{integrations.retention.failure_threshold}
			</span>
		</SettingsRow>
		<SettingsRow
			label={t('retention.logs')}
			description={integrations.journal.configured ? integrations.journal.url : t('retention.logsLocal')}
		>
			<StatusDot
				tone={integrations.journal.configured ? 'success' : 'neutral'}
				label={integrations.journal.configured ? t('retention.journal') : t('retention.localOnly')}
			/>
		</SettingsRow>
	</SettingsSection>
{/if}
