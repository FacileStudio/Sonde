<script lang="ts">
	import { goto } from '$app/navigation';
	import { Card, PageHeader } from '@facile/muse';
	import MonitorForm from '$lib/components/MonitorForm.svelte';
	import { backend, type MonitorInput } from '$lib/backend';

	const blank: MonitorInput = {
		slug: '',
		name: '',
		type: 'http',
		target: '',
		interval_seconds: 60,
		timeout_seconds: 10,
		expected_status: 200,
		expected_keyword: '',
		enabled: true
	};

	let value = $state<MonitorInput>({ ...blank });
	let busy = $state(false);
	let error = $state('');

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (busy) return;
		busy = true;
		error = '';
		try {
			const created = await backend.createMonitor(value);
			goto(`/monitors/${created.id}`);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to create monitor.';
			busy = false;
		}
	}
</script>

<svelte:head>
	<title>New monitor — Sonde</title>
</svelte:head>

<PageHeader title="New monitor" description="Choose what to watch and how often." back={{ href: '/', label: 'Monitors' }} />

<Card class="max-w-2xl p-6">
	<form onsubmit={submit}>
		<MonitorForm {value} submitLabel="Create monitor" {busy} {error} />
	</form>
</Card>
