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
		toast,
		Tooltip
	} from '@facile/muse';
	import StatusPageDrawer from '$lib/components/StatusPageDrawer.svelte';
	import { backend, type AuthUser, type Monitor, type StatusPage } from '$lib/backend';
	import { t } from '$lib/i18n/index.svelte';

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
			error = err instanceof Error ? err.message : t('statusPages.loadFailed');
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
		toast.success(
			known
				? t('statusPages.saved', { title: saved.title })
				: t('statusPages.live', { title: saved.title, slug: saved.slug })
		);
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
			toast.success(t('statusPages.takenDown', { title: existing.title }));
		} catch (err) {
			/* The toast is the whole report. `error` means "the list never loaded" and is the
			   first render branch, so writing a failed delete into it replaces the table the
			   user is still looking at with a load error that did not happen. */
			toast.danger(err instanceof Error ? err.message : t('statusPages.deleteFailed'));
			throw err;
		}
	}
</script>

<svelte:head>
	<title>{t('nav.statusPages')} · Sonde</title>
</svelte:head>

<div class="flex flex-col gap-10">
	<PageHeader
		title={t('nav.statusPages')}
		description={t('statusPages.description')}
	>
		{#snippet actions()}
			{#if isAdmin}
				<Button icon={icons.plus} onclick={openCreate}>{t('statusPages.new')}</Button>
			{/if}
		{/snippet}
	</PageHeader>

	{#if !isAdmin}
		<EmptyState
			icon={icons.shield}
			title={t('statusPages.adminOnly.title')}
			description={t('statusPages.adminOnly.description')}
		/>
	{:else if error}
		<Alert tone="danger" title={t('statusPages.loadError')}>{error}</Alert>
	{:else if loading}
		<div class="flex flex-col gap-2">
			<Skeleton class="h-10 w-full" />
			<Skeleton class="h-10 w-full" />
		</div>
	{:else if pages.length === 0}
		<EmptyState
			icon={icons.globe}
			title={t('statusPages.empty.title')}
			description={t('statusPages.empty.description')}
		>
			<Button variant="secondary" icon={icons.plus} onclick={openCreate}>{t('statusPages.empty.cta')}</Button>
		</EmptyState>
	{:else}
		<Table>
			<thead>
				<tr>
					<th scope="col">{t('statusPages.col.title')}</th>
					<th scope="col">{t('statusPages.col.address')}</th>
					<th scope="col">{t('statusPages.col.monitors')}</th>
					<th scope="col"><span class="sr-only">{t('common.actions')}</span></th>
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
									{t('common.view')}
								</Button>
								<Tooltip text={t('statusPages.editAria', { title: existing.title })} describe={false} class="contents">
									<Button
										variant="ghost"
										size="sm"
										icon={icons.edit}
										aria-label={t('statusPages.editAria', { title: existing.title })}
										onclick={() => openEdit(existing)}
									/>
								</Tooltip>
								<Tooltip text={t('statusPages.deleteAria', { title: existing.title })} describe={false} class="contents">
									<Button
										variant="ghost-danger"
										size="sm"
										icon={icons.remove}
										aria-label={t('statusPages.deleteAria', { title: existing.title })}
										onclick={() => askDelete(existing)}
									/>
								</Tooltip>
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
	title={pendingDelete
		? t('statusPages.delete.title', { title: pendingDelete.title })
		: t('statusPages.delete.titleFallback')}
	description={t('statusPages.delete.description')}
	confirmLabel={t('statusPages.delete.confirm')}
	onConfirm={confirmDelete}
	onCancel={() => (pendingDelete = null)}
/>
