<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { Badge, Card, EmptyState, Spinner, StatusDot } from '@facile/muse';
	import { backend, type PublicStatusPage } from '$lib/backend';

	let data = $state<PublicStatusPage | null>(null);
	let missing = $state(false);

	let slug = $derived(page.params.slug ?? '');

	const overallDown = $derived(data ? data.monitors.some((m) => m.current_status === 'down') : false);
	const bannerTone = $derived(!data ? 'neutral' : overallDown ? 'danger' : 'success');

	onMount(async () => {
		try {
			data = await backend.publicStatusPage(slug);
			document.title = `${data.title} — Status`;
		} catch {
			missing = true;
		}
	});
</script>

<div class="min-h-dvh bg-fc-page px-4 py-10">
	<div class="mx-auto flex w-full max-w-2xl flex-col gap-6">
		{#if missing}
			<EmptyState icon="globe" title="Page not found" description="No public status page lives at this address." />
		{:else if !data}
			<div class="flex justify-center py-16"><Spinner /></div>
		{:else}
			<header class="text-center">
				<h1 class="text-fc-xl font-semibold text-fc-fg">{data.title}</h1>
			</header>

			<div
				class="rounded-md border p-4 text-center text-fc-sm font-medium {bannerTone === 'success'
					? 'border-fc-success/40 bg-fc-success/10 text-fc-success'
					: 'border-fc-danger/40 bg-fc-danger/10 text-fc-danger'}"
			>
				{overallDown ? 'Some systems are down. We are on it.' : 'All systems operational.'}
			</div>

			<div class="flex flex-col gap-3">
				{#each data.monitors as monitor (monitor.id)}
					<Card class="p-4">
						<div class="flex items-center gap-3">
							<StatusDot
								tone={monitor.current_status === 'up' ? 'success' : monitor.current_status === 'down' ? 'danger' : 'neutral'}
								pulse={monitor.current_status === 'down'}
							/>
							<span class="flex-1 truncate font-medium text-fc-fg">{monitor.name}</span>
							<Badge tone="neutral">{monitor.uptime_24h === null ? '—' : `${monitor.uptime_24h.toFixed(2)}%`} · 24h</Badge>
							<Badge tone="neutral">{monitor.uptime_7d === null ? '—' : `${monitor.uptime_7d.toFixed(2)}%`} · 7d</Badge>
						</div>
						{#if monitor.open_incidents.length > 0}
							<ul class="mt-3 flex flex-col gap-1 border-t border-fc-border pt-3">
								{#each monitor.open_incidents as incident (incident.id)}
									<li class="text-fc-xs text-fc-fg-muted">
										{incident.cause} — since {new Date(incident.opened_at).toLocaleString()}
									</li>
								{/each}
							</ul>
						{/if}
					</Card>
				{/each}
			</div>

			<p class="text-center text-fc-xs text-fc-fg-muted">Powered by Sonde</p>
		{/if}
	</div>
</div>
