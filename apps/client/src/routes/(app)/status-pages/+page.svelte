<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { Badge, Button, Card, Checkbox, EmptyState, Field, Input, PageHeader, Spinner } from '@facile/muse';
	import { backend, type Monitor, type StatusPage } from '$lib/backend';

	let pages = $state<StatusPage[] | null>(null);
	let monitors = $state<Monitor[]>([]);
	let error = $state('');
	let busy = $state(false);

	let slug = $state('');
	let title = $state('');
	let picked = $state<number[]>([]);
	let editing = $state<StatusPage | null>(null);

	onMount(async () => {
		try {
			[pages, monitors] = await Promise.all([backend.listStatusPages(), backend.listMonitors()]);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load status pages.';
		}
	});

	function startEdit(existing: StatusPage) {
		editing = existing;
		slug = existing.slug;
		title = existing.title;
		picked = existing.monitors.map((line) => line.id);
	}

	function reset() {
		editing = null;
		slug = '';
		title = '';
		picked = [];
	}

	function toggle(id: number) {
		picked = picked.includes(id) ? picked.filter((v) => v !== id) : [...picked, id];
	}

	function move(id: number, delta: number) {
		const index = picked.indexOf(id);
		const target = index + delta;
		if (index < 0 || target < 0 || target >= picked.length) return;
		const next = [...picked];
		next.splice(index, 1);
		next.splice(target, 0, id);
		picked = next;
	}

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (busy) return;
		busy = true;
		error = '';
		try {
			if (editing) {
				await backend.updateStatusPage(editing.id, { slug, title, monitor_ids: picked });
			} else {
				await backend.createStatusPage({ slug, title, monitor_ids: picked });
			}
			reset();
			pages = await backend.listStatusPages();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to save status page.';
		} finally {
			busy = false;
		}
	}

	async function remove(existing: StatusPage) {
		if (!confirm(`Delete status page "${existing.title}"?`)) return;
		await backend.deleteStatusPage(existing.id).catch(() => undefined);
		pages = await backend.listStatusPages();
	}
</script>

<svelte:head>
	<title>Status pages — Sonde</title>
</svelte:head>

<PageHeader
	title="Status pages"
	description="Public pages showing which of your projects are up."
/>

<div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
	<Card class="p-6">
		<form onsubmit={submit} class="flex flex-col gap-4">
			<h2 class="font-medium text-fc-fg">{editing ? `Edit "${editing.title}"` : 'New status page'}</h2>
			{#if error}
				<div class="rounded-md border border-fc-border bg-fc-danger/10 p-3 text-fc-sm text-fc-danger">
					{error}
				</div>
			{/if}
			<Field label="Title">
				{#snippet children({ id })}
					<Input {id} bind:value={title} placeholder="Acme status" required />
				{/snippet}
			</Field>
			<Field label="Slug">
				{#snippet children({ id })}
					<Input {id} bind:value={slug} placeholder="acme" required />
				{/snippet}
			</Field>
			<div class="flex flex-col gap-2">
				<span class="text-fc-sm font-medium text-fc-fg">Monitors (order matters)</span>
				{#if picked.length > 0}
					<ul class="flex flex-col gap-1">
						{#each picked as id, index (id)}
							{@const monitor = monitors.find((m) => m.id === id)}
							<li class="flex items-center gap-2 rounded border border-fc-border px-2 py-1.5">
								<span class="flex-1 truncate text-fc-sm text-fc-fg">{monitor?.name ?? `#${id}`}</span>
								<Button type="button" variant="ghost" size="sm" disabled={index === 0} onclick={() => move(id, -1)}>↑</Button>
								<Button
									type="button"
									variant="ghost"
									size="sm"
									disabled={index === picked.length - 1}
									onclick={() => move(id, 1)}>↓</Button>
								<Button type="button" variant="ghost" size="sm" onclick={() => toggle(id)}>✕</Button>
							</li>
						{/each}
					</ul>
				{/if}
				{#each monitors.filter((m) => !picked.includes(m.id)) as monitor (monitor.id)}
					<label
					class="flex cursor-pointer items-center gap-2 rounded border border-fc-border px-2 py-1.5
						hover:bg-fc-surface-hover"
				>
						<Checkbox checked={false} onchange={() => toggle(monitor.id)} />
						<span class="text-fc-sm text-fc-fg">{monitor.name}</span>
					</label>
				{/each}
			</div>
			<div class="flex gap-2">
				<Button type="submit" disabled={busy}>{editing ? 'Save changes' : 'Create page'}</Button>
				{#if editing}
					<Button type="button" variant="outline" onclick={reset}>Cancel</Button>
				{/if}
			</div>
		</form>
	</Card>

	<div class="flex flex-col gap-2">
		{#if !pages}
			<div class="flex justify-center py-16"><Spinner /></div>
		{:else if pages.length === 0}
			<EmptyState
				icon="globe"
				title="No public pages yet"
				description="Publish a page so clients can see uptime without an account."
			/>
		{:else}
			{#each pages as existing (existing.id)}
				<Card class="p-4">
					<div class="flex items-center gap-3">
						<div class="min-w-0 flex-1">
							<div class="flex items-center gap-2">
								<span class="truncate font-medium text-fc-fg">{existing.title}</span>
								<Badge tone="neutral">/{existing.slug}</Badge>
							</div>
							<div class="text-fc-xs text-fc-fg-muted">{existing.monitors.length} monitors</div>
						</div>
						<a href={`/status/${existing.slug}`} target="_blank">
							<Button variant="outline" size="sm">View</Button>
						</a>
						<Button variant="ghost" size="sm" onclick={() => startEdit(existing)}>Edit</Button>
						<Button variant="ghost" size="sm" onclick={() => remove(existing)}>✕</Button>
					</div>
				</Card>
			{/each}
			<p class="text-fc-xs text-fc-fg-muted">
				Public URL: <code>{`${window.location.origin}/status/<slug>`}</code>
			</p>
		{/if}
	</div>
</div>
