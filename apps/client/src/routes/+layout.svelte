<script lang="ts">
	import '../app.css';

	void import('iconify-icon');

	let { children } = $props();
</script>

<svelte:head>
	<link rel="icon" href="/favicon.ico" type="image/x-icon" />
</svelte:head>

{@render children()}
