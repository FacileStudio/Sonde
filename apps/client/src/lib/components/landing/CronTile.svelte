<script lang="ts">
	import { fade } from 'svelte/transition';
	import { Badge, StatusDot } from '@facile/muse';
	import { Roll, Stage, Tile, Window, type Act } from '@facile/mise-en-scene';
	import { t } from '$lib/i18n/index.svelte';

	const LIMIT = 26;

	let hours = $state(0);

	const down = $derived(hours >= LIMIT);

	function reset() {
		hours = 0;
	}

	function poster() {
		hours = LIMIT;
	}

	async function script(act: Act) {
		await act.wait(1400);
		await act.count(LIMIT, (value) => (hours = value), 90);
		await act.wait(3000);
		reset();
		await act.wait(1200);
	}
</script>

<Tile title={t('landing.card.cron.title')} body={t('landing.card.cron.body')}>
	<Stage width={320} height={240} {script} {reset} {poster}>
		{#snippet children()}
			<Window style="top:16px;left:20px;width:280px;height:208px">
				<div class="cr-body">
					<div class="cr-row" class:cr-down={down}>
						<StatusDot tone={down ? 'danger' : 'success'} pulse={down} />
						<span class="cr-name">nightly-backup</span>
						<Badge tone="neutral">push</Badge>
					</div>
					<div class="cr-line">0 3 * * * · sonde push</div>
					<div class="cr-state" class:text-fc-danger={down}>
						<Roll value={hours === 0 ? t('landing.demo.lastSignal') : t('landing.demo.noSignal', { hours })} />
					</div>
					<div class="cr-badge">
						{#if down}
							<span transition:fade={{ duration: 220 }}><Badge tone="danger">{t('landing.demo.incident')}</Badge></span>
						{/if}
					</div>
				</div>
			</Window>
		{/snippet}
	</Stage>
</Tile>

<style>
	.cr-body {
		display: flex;
		flex-direction: column;
		gap: 12px;
		padding: 14px;
		color: var(--color-fc-fg);
		font-family: var(--font-fc-body);
	}
	.cr-row {
		display: grid;
		grid-template-columns: auto minmax(0, 1fr) auto;
		align-items: center;
		gap: 10px;
		height: 44px;
		padding: 0 12px;
		border-radius: var(--radius-fc-md);
		background: color-mix(in oklab, var(--color-fc-fg) 4%, transparent);
		transition: background-color 300ms;
	}
	.cr-down {
		background: color-mix(in oklab, var(--color-fc-danger) 14%, transparent);
	}
	.cr-name {
		overflow: hidden;
		font-size: 13px;
		font-weight: 700;
		white-space: nowrap;
		text-overflow: ellipsis;
	}
	.cr-line {
		font-family: var(--font-fc-mono);
		font-size: 11px;
		color: var(--color-fc-fg-muted);
	}
	.cr-state {
		height: 18px;
		font-family: var(--font-fc-mono);
		font-size: 12px;
		white-space: nowrap;
		color: var(--color-fc-fg-muted);
	}
	.cr-badge {
		height: 24px;
	}
</style>
