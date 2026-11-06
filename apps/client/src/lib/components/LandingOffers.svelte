<script lang="ts">
	import { PricingPlans } from '@facile/muse';
	import { t } from '$lib/i18n/index.svelte';
</script>

<PricingPlans
	title={t('landing.pricing')}
	class="mx-auto mb-16 w-[calc(100%-2rem)] max-w-fc-xl rounded-fc-lg bg-fc-component p-6 text-fc-fg sm:p-10"
/>
