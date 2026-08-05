<script lang="ts">
	type Props = { for?: string; children?: import('svelte').Snippet };
	let { for: htmlFor, children }: Props = $props();
</script>

<label class="text-sm font-semibold text-[var(--secondary-foreground)]" for={htmlFor}>
	{@render children?.()}
</label>
