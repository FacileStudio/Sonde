<script lang="ts">
	import { fade } from 'svelte/transition';
	import { Badge, Icon, StatusDot } from '@facile/muse';
	import { Roll, Stage, Tile, Window, type Act } from '@facile/mise-en-scene';
	import { t } from '$lib/i18n/index.svelte';

	type Tick = 'ok' | 'down';

	const TICKS = 14;
	const FILL = { ok: 'var(--color-fc-success)', down: 'var(--color-fc-danger)' };

	let ticks = $state<Tick[]>(Array<Tick>(TICKS).fill('ok'));
	let failures = $state(0);
	let alerted = $state(false);

	const down = $derived(failures >= 3);

	function reset() {
		ticks = Array<Tick>(TICKS).fill('ok');
		failures = 0;
		alerted = false;
	}

	function fail() {
		ticks = [...ticks.slice(1), 'down'];
		failures += 1;
	}

	function poster() {
		reset();
		fail();
		fail();
		fail();
		alerted = true;
	}

	async function script(act: Act) {
		await act.wait(1100);
		for (let i = 0; i < 3; i++) {
			fail();
			await act.wait(700);
		}
		await act.wait(400);
		alerted = true;
		await act.wait(2600);
		alerted = false;
		failures = 0;
		ticks = [...ticks.slice(1), 'ok'];
		await act.wait(1400);
		reset();
		await act.wait(600);
	}
</script>

<Tile title={t('landing.card.down.title')} body={t('landing.card.down.body')}>
	<Stage width={320} height={240} {script} {reset} {poster}>
		{#snippet children()}
			<Window style="top:16px;left:20px;width:280px;height:172px">
				<div class="sd-body">
					<div class="sd-row" class:sd-down={down}>
						<StatusDot tone={down ? 'danger' : 'success'} pulse={down} />
						<span class="sd-name">shop.example.com</span>
						<Badge tone="neutral">http</Badge>
					</div>
					<div class="sd-ticks">
						{#each ticks as tick, i (i)}<span class="sd-tick" style:background={FILL[tick]}></span>{/each}
					</div>
					<div class="sd-state" class:text-fc-danger={failures > 0}>
						<Roll value={down ? t('landing.demo.incident') : failures > 0 ? t('landing.demo.failed', { count: failures }) : '200 OK · 142 ms'} />
					</div>
				</div>
			</Window>
			{#if alerted}
				<div class="sd-alert" transition:fade={{ duration: 260 }}>
					<Icon icon="solar:bell-bing-bold-duotone" size={18} />
					<span>{t('landing.demo.alert')}</span>
				</div>
			{/if}
		{/snippet}
	</Stage>
</Tile>

<style>
	.sd-body {
		display: flex;
		flex-direction: column;
		gap: 12px;
		padding: 14px;
		color: var(--color-fc-fg);
		font-family: var(--font-fc-body);
	}
	.sd-row {
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
	.sd-down {
		background: color-mix(in oklab, var(--color-fc-danger) 14%, transparent);
	}
	.sd-name {
		overflow: hidden;
		font-size: 13px;
		font-weight: 700;
		white-space: nowrap;
		text-overflow: ellipsis;
	}
	.sd-ticks {
		display: flex;
		gap: 3px;
		height: 18px;
	}
	.sd-tick {
		flex: 1;
		border-radius: 2px;
		transition: background-color 300ms;
	}
	.sd-state {
		height: 18px;
		font-family: var(--font-fc-mono);
		font-size: 12px;
		white-space: nowrap;
		color: var(--color-fc-fg-muted);
	}
	.sd-alert {
		position: absolute;
		top: 180px;
		left: 44px;
		display: flex;
		align-items: center;
		gap: 8px;
		width: 232px;
		height: 44px;
		padding: 0 14px;
		border-radius: var(--radius-fc-lg);
		background: var(--color-fc-inverse);
		color: var(--color-fc-on-inverse);
		font-family: var(--font-fc-body);
		font-size: 12px;
		font-weight: 600;
		white-space: nowrap;
		box-shadow: 0 10px 24px -8px rgb(0 0 0 / 0.35);
	}
</style>
