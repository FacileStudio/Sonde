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
		StatusDot,
		Table,
		icons,
		toast
	} from '@facile/muse';
	import MonitorDrawer from '$lib/components/MonitorDrawer.svelte';
	import { backend, type Monitor } from '$lib/backend';

	let monitors = $state<Monitor[]>([]);
	let loading = $state(true);
	let error = $state('');
	let creating = $state(false);
	let pendingDelete = $state<Monitor | null>(null);
	let confirmOpen = $state(false);

	onMount(async () => {
		try {
			monitors = await backend.listMonitors();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load monitors.';
		} finally {
			loading = false;
		}
	});

	function onCreated(monitor: Monitor) {
		monitors = [...monitors, monitor];
		toast.success(`${monitor.name} is being watched.`);
	}

	function askDelete(monitor: Monitor) {
		pendingDelete = monitor;
		confirmOpen = true;
	}

	async function confirmDelete() {
		const monitor = pendingDelete;
		if (!monitor) return;
		try {
			await backend.deleteMonitor(monitor.id);
			monitors = monitors.filter((entry) => entry.id !== monitor.id);
			pendingDelete = null;
			toast.success(`${monitor.name} deleted.`);
		} catch (err) {
			/* The toast is the whole report. `error` means "the list never loaded" and is the
			   first render branch, so writing a failed delete into it replaces the table the
			   user is still looking at with a load error that did not happen. */
			toast.danger(err instanceof Error ? err.message : 'Could not delete the monitor.');
			throw err;
		}
	}

	const targetOf = (monitor: Monitor) =>
		monitor.type === 'push' ? 'Heartbeat endpoint' : monitor.target || '—';
</script>

<svelte:head>
	<title>Monitors — Sonde</title>
</svelte:head>

<div class="flex flex-col gap-10">
	<PageHeader title="Monitors" description="What Sonde watches, and whether it is alive right now.">
		{#snippet actions()}
			<Button icon={icons.plus} onclick={() => (creating = true)}>New monitor</Button>
		{/snippet}
	</PageHeader>

	{#if error}
		<Alert tone="danger" title="Could not load monitors">{error}</Alert>
	{:else if loading}
		<div class="flex flex-col gap-2">
			<Skeleton class="h-10 w-full" />
			<Skeleton class="h-10 w-full" />
			<Skeleton class="h-10 w-full" />
		</div>
	{:else if monitors.length === 0}
		<EmptyState
			icon={icons.monitor}
			title="No monitors yet"
			description="Add an HTTP endpoint, a TCP port, or a push heartbeat to start watching."
		>
			<Button variant="outline" icon={icons.plus} onclick={() => (creating = true)}>
				Create your first monitor
			</Button>
		</EmptyState>
	{:else}
		<Table>
			<thead>
				<tr>
					<th scope="col">Name</th>
					<th scope="col">Type</th>
					<th scope="col">Target</th>
					<th scope="col">Interval</th>
					<th scope="col"><span class="sr-only">Actions</span></th>
				</tr>
			</thead>
			<tbody>
				{#each monitors as monitor (monitor.id)}
					<tr>
						<td>
							<div class="flex min-w-0 items-center gap-2">
								<StatusDot tone={monitor.enabled ? 'success' : 'neutral'} />
								<a
									href={`/monitors/${monitor.id}`}
									class="truncate font-medium text-fc-fg underline underline-offset-4 hover:text-fc-accent"
								>
									{monitor.name}
								</a>
								{#if !monitor.enabled}
									<Badge tone="warning">paused</Badge>
								{/if}
							</div>
						</td>
						<td><Badge tone="neutral">{monitor.type}</Badge></td>
						<td class="max-w-xs truncate text-fc-fg-muted">{targetOf(monitor)}</td>
						<td class="text-fc-fg-muted">{monitor.interval_seconds}s</td>
						<td>
							<div class="flex justify-end gap-1">
								<Button
									variant="ghost"
									size="sm"
									icon={icons.edit}
									aria-label={`Open ${monitor.name}`}
									href={`/monitors/${monitor.id}`}
								/>
								<Button
									variant="ghost-danger"
									size="sm"
									icon={icons.remove}
									aria-label={`Delete ${monitor.name}`}
									onclick={() => askDelete(monitor)}
								/>
							</div>
						</td>
					</tr>
				{/each}
			</tbody>
		</Table>
	{/if}
</div>

<MonitorDrawer bind:open={creating} {onCreated} />

<ConfirmModal
	bind:open={confirmOpen}
	tone="danger"
	title={pendingDelete ? `Delete ${pendingDelete.name}?` : 'Delete monitor?'}
	description="Its check history, uptime record and incidents are deleted with it. This cannot be undone."
	confirmLabel="Delete monitor"
	onConfirm={confirmDelete}
	onCancel={() => (pendingDelete = null)}
/>
