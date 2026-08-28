<script lang="ts">
	import { onMount } from 'svelte';
	import {
		Alert,
		Badge,
		Button,
		ConfirmModal,
		EmptyState,
		PageHeader,
		Skeleton,
		Table,
		icons,
		toast
	} from '@facile/muse';
	import StatusPageDrawer from '$lib/components/StatusPageDrawer.svelte';
	import { backend, type Monitor, type StatusPage } from '$lib/backend';

	let pages = $state<StatusPage[]>([]);
	let monitors = $state<Monitor[]>([]);
	let loading = $state(true);
	let error = $state('');

	let drawerOpen = $state(false);
	let editing = $state<StatusPage | null>(null);
	let pendingDelete = $state<StatusPage | null>(null);
	let confirmOpen = $state(false);

	onMount(async () => {
		try {
			[pages, monitors] = await Promise.all([backend.listStatusPages(), backend.listMonitors()]);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load status pages.';
		} finally {
			loading = false;
		}
	});

	function openCreate() {
		editing = null;
		drawerOpen = true;
	}

	function openEdit(existing: StatusPage) {
		editing = existing;
		drawerOpen = true;
	}

	function onSaved(saved: StatusPage) {
		const known = pages.some((entry) => entry.id === saved.id);
		pages = known
			? pages.map((entry) => (entry.id === saved.id ? saved : entry))
			: [...pages, saved];
		toast.success(known ? `${saved.title} saved.` : `${saved.title} is live at /status/${saved.slug}.`);
		editing = null;
	}

	function askDelete(existing: StatusPage) {
		pendingDelete = existing;
		confirmOpen = true;
	}

	async function confirmDelete() {
		const existing = pendingDelete;
		if (!existing) return;
		try {
			await backend.deleteStatusPage(existing.id);
			pages = pages.filter((entry) => entry.id !== existing.id);
			pendingDelete = null;
			toast.success(`${existing.title} taken down.`);
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Could not delete the status page.';
			toast.danger(message);
			error = message;
			throw err;
		}
	}
</script>

<svelte:head>
	<title>Status pages — Sonde</title>
</svelte:head>

<div class="flex flex-col gap-10">
	<PageHeader
		title="Status pages"
		description="Public pages showing which of your projects are up. No account needed to read them."
	>
		{#snippet actions()}
			<Button icon={icons.plus} onclick={openCreate}>New status page</Button>
		{/snippet}
	</PageHeader>

	{#if error}
		<Alert tone="danger" title="Could not load status pages">{error}</Alert>
	{:else if loading}
		<div class="flex flex-col gap-2">
			<Skeleton class="h-10 w-full" />
			<Skeleton class="h-10 w-full" />
		</div>
	{:else if pages.length === 0}
		<EmptyState
			icon={icons.globe}
			title="No public pages yet"
			description="Publish a page so clients can see uptime without an account."
		>
			<Button variant="outline" icon={icons.plus} onclick={openCreate}>Create a status page</Button>
		</EmptyState>
	{:else}
		<Table>
			<thead>
				<tr>
					<th scope="col">Title</th>
					<th scope="col">Address</th>
					<th scope="col">Monitors</th>
					<th scope="col"><span class="sr-only">Actions</span></th>
				</tr>
			</thead>
			<tbody>
				{#each pages as existing (existing.id)}
					<tr>
						<td class="max-w-xs truncate font-medium text-fc-fg">{existing.title}</td>
						<td><Badge tone="neutral">/status/{existing.slug}</Badge></td>
						<td class="text-fc-fg-muted">{existing.monitors.length}</td>
						<td>
							<div class="flex justify-end gap-1">
								<Button
									variant="ghost"
									size="sm"
									href={`/status/${existing.slug}`}
									target="_blank"
									rel="noopener"
								>
									View
								</Button>
								<Button
									variant="ghost"
									size="sm"
									icon={icons.edit}
									aria-label={`Edit ${existing.title}`}
									onclick={() => openEdit(existing)}
								/>
								<Button
									variant="ghost-danger"
									size="sm"
									icon={icons.remove}
									aria-label={`Delete ${existing.title}`}
									onclick={() => askDelete(existing)}
								/>
							</div>
						</td>
					</tr>
				{/each}
			</tbody>
		</Table>
	{/if}
</div>

<StatusPageDrawer bind:open={drawerOpen} {monitors} {editing} {onSaved} />

<ConfirmModal
	bind:open={confirmOpen}
	tone="danger"
	title={pendingDelete ? `Delete ${pendingDelete.title}?` : 'Delete status page?'}
	description="The public address stops resolving immediately. The monitors it lists are untouched."
	confirmLabel="Delete page"
	onConfirm={confirmDelete}
	onCancel={() => (pendingDelete = null)}
/>
