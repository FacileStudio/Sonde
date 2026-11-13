<script lang="ts">
	import { Alert, Button, Card, Progress, Spinner, StatusDot, icons } from '@facile/muse';
	import type { MaintenanceScan } from '$lib/backend';
	import { t } from '$lib/i18n/index.svelte';

	let { scan }: { scan: MaintenanceScan } = $props();

	let now = $state(Date.now());

	$effect(() => {
		const timer = setInterval(() => (now = Date.now()), 1000);
		return () => clearInterval(timer);
	});

	const elapsed = $derived.by(() => {
		if (!scan.started_at) return '';
		const seconds = Math.max(0, Math.floor((now - new Date(scan.started_at).getTime()) / 1000));
		return `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, '0')}`;
	});

	const stateLabel = $derived(
		scan.state === 'queued'
			? t('maintenance.scan.state.queued')
			: scan.state === 'delivering'
				? t('maintenance.scan.state.delivering')
				: t('maintenance.scan.state.in_progress')
	);

	const progress = $derived(scan.progress);
	const counting = $derived(!!progress && progress.repos_total > 0 && progress.phase !== 'listing');

	type Tone = 'neutral' | 'accent' | 'success' | 'danger';

	function stepTone(step: MaintenanceScan['steps'][number]): Tone {
		if (step.status === 'in_progress') return 'accent';
		if (step.status !== 'completed') return 'neutral';
		if (step.conclusion === 'success') return 'success';
		return step.conclusion === 'skipped' ? 'neutral' : 'danger';
	}
</script>

{#if scan.state === 'failed'}
	<Alert tone="danger" title={t('maintenance.scan.failed.title')}>
		<div class="flex flex-col items-start gap-3">
			<span>{t('maintenance.scan.failed.body', { conclusion: scan.conclusion || scan.state })}</span>
			{#if scan.run_url}
				<Button variant="ghost" size="sm" href={scan.run_url} target="_blank" rel="noopener" iconRight={icons.external}>
					{t('maintenance.scan.openRun')}
				</Button>
			{/if}
		</div>
	</Alert>
{:else}
	<Card class="flex flex-col gap-5" aria-live="polite">
		<div class="flex flex-wrap items-center justify-between gap-3">
			<div class="flex items-center gap-3">
				<Spinner size="sm" label={t('maintenance.scan.title')} />
				<div class="flex flex-col">
					<span class="font-medium text-fc-fg">{t('maintenance.scan.title')}</span>
					<span class="text-sm text-fc-fg-muted">
						{stateLabel}{elapsed ? ` · ${t('maintenance.scan.elapsed', { time: elapsed })}` : ''}
					</span>
				</div>
			</div>
			{#if scan.run_url}
				<Button variant="ghost" size="sm" href={scan.run_url} target="_blank" rel="noopener" iconRight={icons.external}>
					{t('maintenance.scan.openRun')}
				</Button>
			{/if}
		</div>

		{#if progress}
			<div class="flex flex-col gap-2">
				<div class="flex flex-wrap items-baseline justify-between gap-2 text-sm">
					<span class="text-fc-fg">{t(`maintenance.scan.phase.${progress.phase}`)}</span>
					{#if counting}
						<span class="whitespace-nowrap tabular-nums text-fc-fg-muted">
							{t('maintenance.scan.repos', { done: progress.repos_done, total: progress.repos_total })}
						</span>
					{/if}
				</div>
				{#key counting}
					<Progress
						value={counting ? progress.repos_done : null}
						max={counting ? progress.repos_total : 100}
						label={t(`maintenance.scan.phase.${progress.phase}`)}
					/>
				{/key}
				{#if progress.current && progress.phase === 'scanning'}
					<span class="truncate text-sm text-fc-fg-muted">{t('maintenance.scan.current', { name: progress.current })}</span>
				{/if}
			</div>
		{:else}
			<div class="flex flex-col gap-2">
				<span class="text-sm text-fc-fg-muted">{t('maintenance.scan.waiting')}</span>
				<Progress value={null} label={t('maintenance.scan.waiting')} />
			</div>
		{/if}

		{#if scan.steps.length > 0}
			<div class="flex flex-col gap-2">
				<span class="text-xs font-medium uppercase tracking-wide text-fc-fg-muted">{t('maintenance.scan.steps')}</span>
				<ol class="flex flex-col gap-1.5">
					{#each scan.steps as step, index (index)}
						<li class="flex items-center gap-2 text-sm">
							<StatusDot tone={stepTone(step)} pulse={step.status === 'in_progress'} />
							<span class={step.status === 'queued' ? 'text-fc-fg-muted' : 'text-fc-fg'}>{step.name}</span>
						</li>
					{/each}
				</ol>
			</div>
		{/if}
	</Card>
{/if}
