<script lang="ts">
	import { cn } from '$lib/utils';

	type Props = { class?: string; children?: import('svelte').Snippet };
	let { class: className = '', children }: Props = $props();
</script>

<section
	class={cn(
		'rounded-[var(--radius)] border border-[var(--border)] bg-[color:color-mix(in_oklab,var(--card)_88%,white)]/95 p-6 shadow-[0_18px_70px_-35px_rgba(85,55,25,0.45)] backdrop-blur',
		className
	)}
>
	{@render children?.()}
</section>
