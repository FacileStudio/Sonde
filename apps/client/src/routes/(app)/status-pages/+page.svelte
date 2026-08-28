<script lang="ts">
	import { getContext, onMount } from 'svelte';
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
	import { backend, type AuthUser, type Monitor, type StatusPage } from '$lib/backend';

	const app = getContext<{ user: AuthUser | null }>('app');

	/* Every status-page route on the API is behind requireAdmin. Fetching as a member gets a
	   raw "administrator role required", so say it here instead of asking and being refused. */
	const isAdmin = $derived(Boolean(app.user?.is_admin));

	let pages = $state<StatusPage[]>([]);
	let monitors = $state<Monitor[]>([]);
	let loading = $state(true);
	let error = $state('');

	let drawerOpen = $state(false);
	let editing = $state<StatusPage | null>(null);
	let pendingDelete = $state<StatusPage | null>(null);
	let confirmOpen = $state(false);

	onMount(async () => {
		if (!isAdmin) {
			loading = false;
			return;
		}
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
			/* The toast is the whole report. `error` means "the list never loaded" and is the
			   first render branch, so writing a failed delete into it replaces the table the
			   user is still looking at with a load error that did not happen. */
			toast.danger(err instanceof Error ? err.message : 'Could not delete the status page.');
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
			{#if isAdmin}
				<Button icon={icons.plus} onclick={openCreate}>New status page</Button>
			{/if}
		{/snippet}
	</PageHeader>

	{#if !isAdmin}
		<EmptyState
			icon={icons.shield}
			title="Administrators only"
			description="Status pages are managed by whoever runs this instance. Ask them for access, or for a link to a page that is already published."
		/>
	{:else if error}
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
