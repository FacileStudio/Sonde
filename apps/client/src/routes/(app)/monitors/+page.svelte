<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
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
	import MonitorStatusChart from '$lib/components/MonitorStatusChart.svelte';
	import { backend, type Check, type Monitor } from '$lib/backend';

	let monitors = $state<Monitor[]>([]);
	let checksByMonitor = $state<Record<number, Check[]>>({});
	let loading = $state(true);
	let error = $state('');
	let creating = $state(false);
	let pendingDelete = $state<Monitor | null>(null);
	let confirmOpen = $state(false);

	onMount(async () => {
		try {
			monitors = await backend.listMonitors();
			void loadChecks();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load monitors.';
		} finally {
			loading = false;
		}
	});

	async function loadChecks() {
		const entries = await Promise.all(
			monitors.map(async (m) => {
				try {
					const checks = await backend.monitorChecks(m.id, 24);
					return [m.id, checks] as const;
				} catch {
					return [m.id, []] as const;
				}
			})
		);
		checksByMonitor = Object.fromEntries(entries);
	}

	function onCreated(monitor: Monitor) {
		monitors = [...monitors, monitor];
		checksByMonitor[monitor.id] = [];
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
					<th scope="col" class="w-48">Recent Probes</th>
					<th scope="col">Type</th>
					<th scope="col">Target</th>
					<th scope="col">Interval</th>
					<th scope="col"><span class="sr-only">Actions</span></th>
				</tr>
			</thead>
			<tbody>
				{#each monitors as monitor (monitor.id)}
					<tr
						class="group cursor-pointer transition-colors hover:bg-fc-surface-subtle"
						onclick={(e) => {
							const target = e.target as HTMLElement;
							if (target.closest('button, a')) return;
							goto(`/monitors/${monitor.id}`);
						}}
					>
						<td>
							<div class="flex min-w-0 items-center gap-2">
								<StatusDot tone={monitor.enabled ? 'success' : 'neutral'} />
								<span
									class="truncate font-medium text-fc-fg group-hover:text-fc-brand transition-colors"
								>
									{monitor.name}
								</span>
								{#if !monitor.enabled}
									<Badge tone="warning">paused</Badge>
								{/if}
							</div>
						</td>
						<td class="w-48">
							<MonitorStatusChart
								checks={checksByMonitor[monitor.id] ?? []}
								slots={24}
								height={16}
							/>
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
