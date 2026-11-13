<script lang="ts">
	import { onMount } from 'svelte';
	import { Alert, Badge, Button, Section, Select, Skeleton, icons, toast } from '@facile/muse';
	import { backend, type Monitor, type ProjectRef } from '$lib/backend';
	import { t } from '$lib/i18n/index.svelte';

	let { monitor, onChange }: { monitor: Monitor; onChange: (monitor: Monitor) => void } = $props();

	let projects = $state<ProjectRef[] | null>(null);
	let reason = $state('');
	let choice = $state('');
	let busy = $state(false);

	const options = $derived((projects ?? []).map((project) => ({ value: String(project.id), label: project.name })));

	/* A 412 carries the reason in plain words: GitHub not configured, or no GitHub account linked
	   in Registre. It is shown as is rather than replaced with a generic failure. */
	onMount(async () => {
		try {
			projects = await backend.listProjects();
		} catch (err) {
			reason = err instanceof Error ? err.message : t('monitor.project.failed');
		}
	});

	async function link(projectId: number | null) {
		if (busy) return;
		busy = true;
		try {
			onChange(await backend.linkMonitorProject(monitor.id, projectId));
			choice = '';
			toast.success(t(projectId === null ? 'monitor.project.unlinked' : 'monitor.project.linked'));
		} catch (err) {
			toast.danger(err instanceof Error ? err.message : t('monitor.project.failed'));
		} finally {
			busy = false;
		}
	}

	/* The picked value is read from the bound state on submit, never from a change event. */
	function submit(event: SubmitEvent) {
		event.preventDefault();
		if (choice) link(Number(choice));
	}
</script>

<Section title={t('monitor.project.title')} description={t('monitor.project.description')}>
	<div class="flex flex-col gap-4">
		{#if monitor.project}
			<div class="flex flex-wrap items-center gap-3">
				<Badge tone="accent">{monitor.project.name}</Badge>
				<a
					href="https://github.com/FacileStudio/{monitor.project.name}"
					target="_blank"
					rel="noopener noreferrer"
					class="text-fc-sm font-medium text-fc-fg underline underline-offset-4 hover:text-fc-accent"
				>
					{t('monitor.project.github')}
				</a>
				<Button variant="ghost" size="sm" icon={icons.close} disabled={busy} onclick={() => link(null)}>
					{t('monitor.project.unlink')}
				</Button>
			</div>
		{:else}
			<p class="text-fc-sm text-fc-fg-muted">{t('monitor.project.none')}</p>
		{/if}

		{#if reason}
			<Alert tone="warning" title={t('monitor.project.unavailable')}>{reason}</Alert>
		{:else if projects === null}
			<Skeleton class="h-11 w-full" />
		{:else if projects.length === 0}
			<p class="text-fc-sm text-fc-fg-muted">{t('monitor.project.empty')}</p>
		{:else}
			<form class="flex flex-wrap items-center gap-3" onsubmit={submit}>
				<Select
					bind:value={choice}
					{options}
					placeholder={t('monitor.project.pick')}
					aria-label={t('monitor.project.pick')}
					wrapperClass="min-w-56 flex-1"
				/>
				<Button type="submit" disabled={busy || !choice}>{t('monitor.project.link')}</Button>
			</form>
		{/if}
	</div>
</Section>
