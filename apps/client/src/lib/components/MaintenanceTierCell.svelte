<script lang="ts">
	import { untrack } from 'svelte';
	import { Select } from '@facile/muse';
	import { backend, MAINTENANCE_TIERS, type MaintenanceProject, type MaintenanceTier } from '$lib/backend';
	import { t } from '$lib/i18n/index.svelte';

	let {
		project,
		onSaved,
		onError
	}: { project: MaintenanceProject; onSaved: () => void; onError: (message: string) => void } = $props();

	const initial = untrack(() => project.tier);
	let value = $state<string>(initial);
	let saved = $state<string>(initial);
	let saving = $state(false);

	const options = $derived(MAINTENANCE_TIERS.map((tier) => ({ value: tier, label: t(`maintenance.tier.${tier}`) })));

	/* One request at a time: the Select is disabled while a save is in flight, so a rollback only ever
	   undoes the change that failed, and no later choice can be overwritten by an earlier response. */
	$effect(() => {
		const next = value;
		if (untrack(() => saving || next === saved)) return;
		saving = true;
		backend
			.setProjectTier(project.id, next as MaintenanceTier)
			.then(() => {
				saved = next;
				onSaved();
			})
			.catch(() => {
				value = untrack(() => saved);
				onError(t('maintenance.tierFailed'));
			})
			.finally(() => (saving = false));
	});
</script>

<Select bind:value {options} disabled={saving} aria-label={t('maintenance.tierLabel', { name: project.name })} />
