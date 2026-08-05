<script lang="ts">
	import { cva, type VariantProps } from 'class-variance-authority';
	import { cn } from '$lib/utils';

	const buttonVariants = cva(
		'inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-[calc(var(--radius)-0.25rem)] text-sm font-semibold transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--ring)] disabled:pointer-events-none disabled:opacity-50',
		{
			variants: {
				variant: {
					default: 'bg-[var(--primary)] text-[var(--primary-foreground)] shadow-sm hover:brightness-95',
					secondary:
						'bg-[var(--secondary)] text-[var(--secondary-foreground)] shadow-sm hover:bg-[var(--muted)]',
					outline:
						'border border-[var(--border)] bg-[var(--card)] text-[var(--foreground)] hover:bg-[var(--muted)]',
					ghost: 'text-[var(--foreground)] hover:bg-[var(--muted)]',
					destructive:
						'bg-[var(--destructive)] text-[var(--destructive-foreground)] shadow-sm hover:brightness-95'
				},
				size: {
					default: 'h-10 px-4 py-2',
					sm: 'h-9 rounded-xl px-3',
					lg: 'h-11 rounded-xl px-6'
				}
			},
			defaultVariants: {
				variant: 'default',
				size: 'default'
			}
		}
	);

	type Props = VariantProps<typeof buttonVariants> & {
		type?: 'button' | 'submit' | 'reset';
		class?: string;
		disabled?: boolean;
		onclick?: () => void;
		children?: import('svelte').Snippet;
	};

	let {
		variant = 'default',
		size = 'default',
		type = 'button',
		class: className = '',
		disabled = false,
		onclick,
		children
	}: Props = $props();
</script>

<button class={cn(buttonVariants({ variant, size }), className)} {type} {disabled} {onclick}>
	{@render children?.()}
</button>
