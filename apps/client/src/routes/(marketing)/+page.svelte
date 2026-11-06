<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import LandingOffers from '$lib/components/LandingOffers.svelte';
	import { Button } from '@facile/muse';
	import { backend } from '$lib/backend';

	let checking = $state(true);

	const features = [
		{
			icon: 'solar:monitor-linear',
			title: 'HTTP, TCP and push monitors',
			text: 'Probe URLs for status and keywords, check ports, let cron jobs heartbeat — silence is an incident.'
		},
		{
			icon: 'solar:chart-2-linear',
			title: 'Uptime that adds up',
			text: 'Rollup aggregates answer 24h to 90d uptime instantly, without a database the size of a planet.'
		},
		{
			icon: 'solar:global-linear',
			title: 'Public status pages',
			text: 'Publish per-project status pages so clients see green without asking you.'
		},
		{
			icon: 'solar:danger-triangle-linear',
			title: 'Alerts through Antenne',
			text: 'Incidents flow straight into the Facile alert bus. No notification zoo included, on purpose.'
		}
	];

	onMount(async () => {
		try {
			await backend.me();
			goto('/monitors');
		} catch {
			checking = false;
		}
	});
</script>

<svelte:head>
	<title>Sonde — self-hosted uptime monitoring</title>
	<meta name="description" content="Watch your services, alert your team, publish your status. Self-hosted uptime monitoring for the Facile Suite." />
</svelte:head>

{#if checking}
	<div class="flex min-h-dvh items-center justify-center bg-fc-page"><span class="text-fc-fg-muted">…</span></div>
{:else}
	<div class="flex min-h-dvh flex-col bg-black text-white">
		<header class="flex items-center justify-between px-6 py-5 sm:px-12">
			<a href="/" class="flex items-center gap-3">
				<iconify-icon icon="solar:radar-2-bold-duotone" width="26" height="26" class="block shrink-0"></iconify-icon>
				<span class="font-fc-title text-lg font-semibold tracking-tight">Sonde</span>
			</a>
			<a href="/login" class="rounded-md border border-white/20 px-4 py-1.5 text-sm transition-colors hover:bg-white/10">
				Log in
			</a>
		</header>

		<main class="mx-auto flex w-full max-w-5xl flex-1 flex-col justify-center px-6 py-16 sm:px-12">
			<h1 class="font-fc-title text-4xl leading-tight font-semibold tracking-tight sm:text-6xl">
				Is it up?<br /><span class="text-white/40">Know in a minute.</span>
			</h1>
			<p class="mt-6 max-w-xl text-base leading-relaxed text-white/50">
				Sonde watches your HTTP endpoints, TCP ports and cron jobs, opens incidents when they
				stop answering, and publishes status pages your clients can read themselves.
				Self-hosted, one binary, part of the Facile Suite.
			</p>
			<div class="mt-8 flex flex-wrap gap-3">
				<a
					href="/login?tab=register"
					class="inline-flex h-11 items-center rounded-md bg-white px-6 text-sm font-medium text-black transition-opacity hover:opacity-90"
				>
					Create account
				</a>
				<a
					href="/login"
					class="inline-flex h-11 items-center rounded-md border border-white/25 px-6 text-sm font-medium transition-colors hover:bg-white/10"
				>
					Log in
				</a>
			</div>

			<div class="mt-20 grid grid-cols-1 gap-8 sm:grid-cols-2">
				{#each features as feature (feature.title)}
					<div>
						<iconify-icon icon={feature.icon} width="22" height="22" class="mb-3 block text-white/70"></iconify-icon>
						<h2 class="text-sm font-semibold">{feature.title}</h2>
						<p class="mt-1.5 text-sm leading-relaxed text-white/40">{feature.text}</p>
					</div>
				{/each}
			</div>
		</main>

		<LandingOffers />

		<footer class="px-6 pb-8 sm:px-12">
			<p class="text-xs text-white/30">© {new Date().getFullYear()} Sonde by Facile.</p>
		</footer>
	</div>
{/if}
