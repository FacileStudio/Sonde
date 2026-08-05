<script lang="ts">
	import { cn } from '$lib/utils';

	type Props = {
		value?: string;
		type?: string;
		placeholder?: string;
		class?: string;
		id?: string;
		required?: boolean;
	};

	let {
		value = $bindable(''),
		type = 'text',
		placeholder = '',
		class: className = '',
		id,
		required = false
	}: Props = $props();
</script>

<input
	bind:value
	{type}
	{placeholder}
	{id}
	{required}
	class={cn(
		'flex h-11 w-full rounded-[calc(var(--radius)-0.3rem)] border border-[var(--input)] bg-white/85 px-3 py-2 text-sm text-[var(--foreground)] shadow-inner outline-none transition focus-visible:ring-2 focus-visible:ring-[var(--ring)]',
		className
	)}
/>
