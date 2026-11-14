<script lang="ts">
	import { StatusDot } from '@facile/muse';
	import { Roll, Stage, Tile, Window, type Act } from '@facile/mise-en-scene';
	import { t } from '$lib/i18n/index.svelte';

	const ROWS = ['landing.demo.rowSite', 'landing.demo.rowShop', 'landing.demo.rowBooking'] as const;
	const BROKEN = 1;

	let down = $state(false);

	function reset() {
		down = false;
	}

	function poster() {
		down = true;
	}

	async function script(act: Act) {
		await act.wait(1600);
		down = true;
		await act.wait(3200);
		down = false;
		await act.wait(2200);
		reset();
		await act.wait(400);
	}
</script>

<Tile title={t('landing.card.status.title')} body={t('landing.card.status.body')}>
	<Stage width={320} height={240} {script} {reset} {poster}>
		{#snippet children()}
			<Window style="top:12px;left:20px;width:280px;height:216px">
				<div class="sp-body">
					<div class="sp-head">
						<span class="sp-title">Atelier Lune</span>
						<span class="sp-url">/status/atelier-lune</span>
					</div>
					<div class="sp-verdict" class:sp-bad={down}>
						<Roll value={down ? t('public.verdict.down') : t('public.verdict.ok')} />
					</div>
					<div class="sp-list">
						{#each ROWS as key, i (key)}
							{@const off = down && i === BROKEN}
							<div class="sp-row">
								<StatusDot tone={off ? 'danger' : 'success'} pulse={off} />
								<span>{t(key)}</span>
							</div>
						{/each}
					</div>
				</div>
			</Window>
		{/snippet}
	</Stage>
</Tile>

<style>
	.sp-body {
		display: flex;
		flex-direction: column;
		gap: 8px;
		padding: 10px 14px;
		color: var(--color-fc-fg);
		font-family: var(--font-fc-body);
	}
	.sp-head {
		display: flex;
		align-items: baseline;
		justify-content: space-between;
		gap: 8px;
		height: 20px;
	}
	.sp-title {
		font-family: var(--font-fc-title);
		font-size: 14px;
		font-weight: 700;
	}
	.sp-url {
		font-family: var(--font-fc-mono);
		font-size: 10px;
		color: var(--color-fc-fg-muted);
	}
	.sp-verdict {
		height: 32px;
		padding: 0 12px;
		border-radius: var(--radius-fc-md);
		background: color-mix(in oklab, var(--color-fc-success) 16%, transparent);
		font-size: 12px;
		font-weight: 700;
		line-height: 32px;
		white-space: nowrap;
		transition: background-color 300ms;
	}
	.sp-bad {
		background: color-mix(in oklab, var(--color-fc-danger) 16%, transparent);
	}
	.sp-list {
		display: grid;
		gap: 4px;
	}
	.sp-row {
		display: flex;
		align-items: center;
		gap: 10px;
		height: 26px;
		padding: 0 4px;
		font-size: 12px;
	}
</style>
