<script lang="ts">
	import {
		Alert,
		Button,
		Drawer,
		Field,
		Icon,
		IconButton,
		Input,
		Spinner,
		icons,
		toast
	} from '@facile/muse';
	import { backend, type Monitor, type StatusPage } from '$lib/backend';

	let {
		open = $bindable(false),
		monitors = [],
		editing = null,
		onSaved
	}: {
		open?: boolean;
		monitors?: Monitor[];
		editing?: StatusPage | null;
		onSaved: (page: StatusPage) => void;
	} = $props();

	let title = $state('');
	let slug = $state('');
	let slugEdited = $state(false);
	let picked = $state<number[]>([]);
	let error = $state('');
	let submitting = $state(false);

	const slugify = (value: string) =>
		value
			.toLowerCase()
			.replace(/[^a-z0-9]+/g, '-')
			.replace(/^-+|-+$/g, '')
			.slice(0, 60);

	const effectiveSlug = $derived(slugEdited || editing ? slug : slugify(title));
	const available = $derived(monitors.filter((monitor) => !picked.includes(monitor.id)));
	const ready = $derived(Boolean(title.trim()) && Boolean(effectiveSlug));

	/* Reopening must never show the previous attempt's error or the monitors it half-picked. */
	$effect(() => {
		if (!open) {
			title = '';
			slug = '';
			slugEdited = false;
			picked = [];
			error = '';
			return;
		}
		if (editing) {
			title = editing.title;
			slug = editing.slug;
			picked = editing.monitors.map((line) => line.id);
		}
	});

	function toggle(id: number) {
		picked = picked.includes(id) ? picked.filter((value) => value !== id) : [...picked, id];
	}

	function move(id: number, delta: number) {
		const index = picked.indexOf(id);
		const target = index + delta;
		if (index < 0 || target < 0 || target >= picked.length) return;
		const next = [...picked];
		next.splice(index, 1);
		next.splice(target, 0, id);
		picked = next;
	}

	const nameOf = (id: number) => monitors.find((monitor) => monitor.id === id)?.name ?? `#${id}`;

	async function save(event: SubmitEvent) {
		event.preventDefault();
		if (submitting) return;
		error = '';
		submitting = true;
		const input = { slug: effectiveSlug, title: title.trim(), monitor_ids: picked };
		try {
			const saved = editing
				? await backend.updateStatusPage(editing.id, input)
				: await backend.createStatusPage(input);
			open = false;
			onSaved(saved);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Could not save the status page.';
			/* The Alert lives inside the sheet. A toast survives it being closed. */
			toast.danger(error);
		} finally {
			submitting = false;
		}
	}
</script>

<Drawer
	bind:open
	title={editing ? 'Edit status page' : 'New status page'}
	description="Anyone with the link sees these monitors and their uptime, in this order."
	showClose
	dismissible={!submitting}
>
	<form id="status-page-form" class="flex flex-col gap-4" onsubmit={save}>
		<Field label="Title" helper="Shown as the heading of the public page.">
			<Input bind:value={title} placeholder="Acme status" required disabled={submitting} />
		</Field>

		<Field label="Slug" helper="The public address is /status/ followed by this.">
			<Input
				value={effectiveSlug}
				oninput={(event: Event) => {
					slugEdited = true;
					slug = (event.currentTarget as HTMLInputElement).value;
				}}
				placeholder="acme"
				required
				disabled={submitting}
			/>
		</Field>

		<div class="flex flex-col gap-2">
			<span class="text-fc-sm font-medium text-fc-fg">Monitors</span>
			{#if picked.length > 0}
				<ul class="flex flex-col gap-1">
					{#each picked as id, index (id)}
						<!-- IconButton, not a size="sm" Button: a drawer is touch-primary and the
						     CHARTE §7 floor for an icon-only control is 44px. -->
						<li class="flex min-w-0 items-center gap-1 rounded-fc-sm bg-fc-surface px-2 py-1">
							<span class="min-w-0 flex-1 truncate text-fc-sm text-fc-fg">{nameOf(id)}</span>
							<IconButton
								variant="ghost"
								aria-label={`Move ${nameOf(id)} up`}
								disabled={index === 0 || submitting}
								onclick={() => move(id, -1)}
							>
								<Icon icon={icons.chevronUp} size={18} />
							</IconButton>
							<IconButton
								variant="ghost"
								aria-label={`Move ${nameOf(id)} down`}
								disabled={index === picked.length - 1 || submitting}
								onclick={() => move(id, 1)}
							>
								<Icon icon={icons.chevronDown} size={18} />
							</IconButton>
							<IconButton
								variant="danger"
								aria-label={`Remove ${nameOf(id)} from this page`}
								disabled={submitting}
								onclick={() => toggle(id)}
							>
								<Icon icon={icons.close} size={18} />
							</IconButton>
						</li>
					{/each}
				</ul>
			{/if}

			{#if available.length > 0}
				<div class="flex flex-wrap gap-2">
					{#each available as monitor (monitor.id)}
						<Button
							variant="outline"
							size="lg"
							icon={icons.plus}
							disabled={submitting}
							onclick={() => toggle(monitor.id)}
						>
							{monitor.name}
						</Button>
					{/each}
				</div>
			{:else if picked.length === 0}
				<p class="flex items-center gap-2 text-fc-sm text-fc-fg-muted">
					<Icon icon={icons.monitor} size={16} />
					No monitors to publish yet.
				</p>
			{/if}
		</div>

		{#if error}
			<Alert tone="danger" title={editing ? 'Not saved' : 'Not added'}>{error}</Alert>
		{/if}
	</form>

	{#snippet footer()}
		<div class="flex gap-2">
			<Button
				variant="ghost"
				size="lg"
				class="flex-1"
				disabled={submitting}
				onclick={() => (open = false)}
			>
				Cancel
			</Button>
			<Button
				type="submit"
				form="status-page-form"
				size="lg"
				icon={editing ? icons.check : icons.plus}
				class="flex-1"
				disabled={submitting || !ready}
			>
				{#if submitting}<Spinner size="sm" />{/if}
				{submitting ? 'Saving…' : editing ? 'Save changes' : 'Create page'}
			</Button>
		</div>
	{/snippet}
</Drawer>
