<script lang="ts">
	import { untrack } from 'svelte';
	import { Select } from '@facile/muse';
	import { backend, MAINTENANCE_TIERS, type MaintenanceProject, type MaintenanceTier } from '$lib/backend';
	import { t } from '$lib/i18n/index.svelte';

	let { project, onError }: { project: MaintenanceProject; onError: (message: string) => void } = $props();

	const initial = untrack(() => project.tier);
	let value = $state<string>(initial);
	let saved = $state<string>(initial);

	const options = $derived(MAINTENANCE_TIERS.map((tier) => ({ value: tier, label: t(`maintenance.tier.${tier}`) })));

	$effect(() => {
		const next = value;
		if (next === saved) return;
		backend
			.setProjectTier(project.id, next as MaintenanceTier)
			.then(() => (saved = next))
			.catch(() => {
				value = saved;
				onError(t('maintenance.tierFailed'));
			});
	});
</script>

<Select bind:value {options} aria-label={t('maintenance.tierLabel', { name: project.name })} />
