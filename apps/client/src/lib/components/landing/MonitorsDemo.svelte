<script lang="ts">
	import { fade } from 'svelte/transition';
	import { Badge, StatusDot } from '@facile/muse';
	import { Bar, Caret, Roll, Stage, Terminal, Window, type Act } from '@facile/mise-en-scene';
	import { t } from '$lib/i18n/index.svelte';

	type State = 'pending' | 'up' | 'down';
	type Tick = 'ok' | 'down' | 'empty';
	type Monitor = { name: string; type: string; state: State; latency: number; ticks: Tick[] };

	const TICKS = 24;
	const PUSH_TOKEN = 'sonde push 7f3a9c2e';
	const SEED = [
		{ name: 'shop.example.com', type: 'http', latency: 142 },
		{ name: 'api.example.com', type: 'http', latency: 87 },
		{ name: 'db.internal:5432', type: 'tcp', latency: 4 },
		{ name: 'nightly-backup', type: 'push', latency: 0 }
	];
	const TONE = { pending: 'neutral', up: 'success', down: 'danger' } as const;
	const FILL = { ok: 'var(--color-fc-success)', down: 'var(--color-fc-danger)', empty: 'var(--color-fc-field)' };
	const INCIDENT = 1;
	const HEARTBEAT = 3;

	let monitors = $state<Monitor[]>(fresh());
	let command = $state('');
	let sent = $state(false);

	const up = $derived(monitors.filter((m) => m.state === 'up').length);
	const open = $derived(monitors[INCIDENT].state === 'down');

	function fresh(): Monitor[] {
		return SEED.map((m) => ({ ...m, state: 'pending', ticks: Array<Tick>(TICKS).fill('empty') }));
	}

	function reset() {
		monitors = fresh();
		command = '';
		sent = false;
	}

	function settle(monitor: Monitor) {
		monitor.state = 'up';
		monitor.ticks = Array<Tick>(TICKS).fill('ok');
	}

	function push(monitor: Monitor, tick: Tick) {
		monitor.ticks = [...monitor.ticks.slice(1), tick];
	}

	function poster() {
		reset();
		monitors.forEach(settle);
		command = PUSH_TOKEN;
		sent = true;
	}

	async function script(act: Act) {
		await act.wait(700);
		for (const monitor of monitors) {
			settle(monitor);
			await act.wait(320);
		}
		await act.wait(900);
		await act.type(PUSH_TOKEN, (v) => (command = v), 45);
		await act.wait(300);
		sent = true;
		push(monitors[HEARTBEAT], 'ok');
		await act.wait(1500);
		monitors[INCIDENT].state = 'down';
		push(monitors[INCIDENT], 'down');
		await act.wait(2600);
		monitors[INCIDENT].state = 'up';
		push(monitors[INCIDENT], 'ok');
		await act.wait(2400);
		reset();
		await act.wait(500);
	}

	function latency(monitor: Monitor) {
		if (monitor.state === 'pending') return '–';
		if (monitor.state === 'down') return t('landing.demo.timeout');
		if (monitor.type === 'push') return sent ? t('common.justNow') : '–';
		return `${monitor.latency} ms`;
	}
</script>

{#snippet nav()}
	<div class="flex items-center gap-2 px-5 pt-3 pb-5 font-fc-title text-[15px] font-bold text-fc-fg">
		<iconify-icon icon="solar:radar-bold-duotone" width="20" height="20"></iconify-icon>
		<span>Sonde</span>
	</div>
	<div class="grid gap-1 px-3 text-[13px]">
		{#each [t('nav.monitors'), t('nav.incidents'), t('nav.statusPages')] as label, i (label)}
			<div class="flex h-[34px] items-center gap-2.5 rounded-fc-md px-2.5 {i === 0 ? 'bg-fc-fg/5 font-bold' : 'text-fc-fg-muted'}">
				<span class="size-4 shrink-0 rounded-[5px] bg-fc-fg/20"></span>{label}
			</div>
		{/each}
	</div>
	<div class="grid gap-3 px-5 pt-6">
		<Bar w="70%" />
		<Bar w="52%" />
	</div>
{/snippet}

{#snippet header(compact: boolean)}
	<div class="flex items-center justify-between gap-3">
		<div class="flex min-w-0 items-center gap-2">
			<span class="font-fc-title text-fc-lg font-bold">{t('nav.monitors')}</span>
			{#if open}
				<span transition:fade={{ duration: 220 }}>
					<Badge tone="danger">{t('monitors.openBadge', { count: 1 })}</Badge>
				</span>
			{/if}
		</div>
		<span class="md-summary" style:width={compact ? '96px' : '120px'}>
			<Roll value={t('landing.demo.summary', { up, total: monitors.length })} />
		</span>
	</div>
{/snippet}

<Stage
	width={960}
	height={480}
	compact={{ below: 640, width: 358, height: 480 }}
	label={t('landing.demo.label')}
	{script}
	{reset}
	{poster}
>
	{#snippet children({ compact })}
		<Window
			style={compact ? 'top:12px;left:12px;width:334px;height:340px' : 'top:16px;left:16px;width:760px;height:420px'}
			sidebar={compact ? undefined : nav}
		>
			<div class="md-body" class:md-compact={compact}>
				{@render header(compact)}
				<div class="md-list">
					{#each monitors as monitor (monitor.name)}
						<div class="md-row" class:md-row-down={monitor.state === 'down'}>
							<StatusDot tone={TONE[monitor.state]} pulse={monitor.state === 'down'} />
							<div class="min-w-0">
								<div class="truncate text-[13px] font-bold">{monitor.name}</div>
								{#if compact}<div class="text-[11px] text-fc-fg-muted">{monitor.type}</div>{/if}
							</div>
							{#if !compact}
								<Badge tone="neutral">{monitor.type}</Badge>
								<div class="md-ticks">
									{#each monitor.ticks as tick, i (i)}
										<span class="md-tick" style:background={FILL[tick]}></span>
									{/each}
								</div>
							{/if}
							<span class="md-latency" class:text-fc-danger={monitor.state === 'down'}>
								<Roll value={latency(monitor)} />
							</span>
						</div>
					{/each}
				</div>
			</div>
		</Window>

		<Terminal style={compact ? 'top:336px;left:24px;width:310px;height:132px' : 'top:330px;left:560px;width:384px;height:124px'}>
			<div class="mb-3 flex items-center gap-2 text-fc-on-inverse-muted">
				<span class="size-2 rounded-fc-pill bg-fc-success"></span>crontab · 0 3 * * *
			</div>
			<div class="whitespace-nowrap">
				<span class="md-dim">$</span>
				{command}{#if !sent}<Caret />{/if}
			</div>
			{#if sent}
				<div class="md-ok whitespace-nowrap" in:fade={{ duration: 160 }}>✓ heartbeat sent</div>
				<div class="whitespace-nowrap" in:fade={{ duration: 160, delay: 200 }}>
					<span class="md-dim">$</span>
					<Caret />
				</div>
			{/if}
		</Terminal>
	{/snippet}
</Stage>

<style>
	.md-body {
		display: flex;
		flex-direction: column;
		gap: 18px;
		padding: 22px 24px;
		font-family: var(--font-fc-body);
	}
	.md-compact {
		gap: 14px;
		padding: 16px;
	}
	.md-summary {
		text-align: right;
		font-family: var(--font-fc-mono);
		font-size: 12px;
		color: var(--color-fc-fg-muted);
	}
	.md-list {
		display: grid;
		gap: 6px;
	}
	.md-row {
		display: grid;
		grid-template-columns: auto minmax(0, 1fr) 52px 140px 72px;
		align-items: center;
		gap: 12px;
		height: 52px;
		padding: 0 14px;
		border-radius: var(--radius-fc-md);
		background: color-mix(in oklab, var(--color-fc-fg) 4%, transparent);
		transition: background-color 300ms;
	}
	.md-compact .md-row {
		grid-template-columns: auto minmax(0, 1fr) 96px;
		gap: 10px;
		height: 50px;
	}
	.md-row-down {
		background: color-mix(in oklab, var(--color-fc-danger) 14%, transparent);
	}
	.md-ticks {
		display: flex;
		align-items: flex-end;
		gap: 2px;
		height: 20px;
	}
	.md-tick {
		flex: 1;
		height: 100%;
		border-radius: 2px;
		transition: background-color 300ms;
	}
	.md-latency {
		text-align: right;
		font-family: var(--font-fc-mono);
		font-size: 12px;
		color: var(--color-fc-fg-muted);
	}
	.md-dim {
		color: var(--color-fc-on-inverse-muted);
	}
	.md-ok {
		color: color-mix(in oklab, var(--color-fc-success), var(--color-fc-on-inverse) 35%);
	}
</style>
