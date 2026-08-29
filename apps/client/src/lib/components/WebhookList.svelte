<script lang="ts">
	import { onMount } from 'svelte';
	import {
		Button,
		ConfirmModal,
		Field,
		Icon,
		IconButton,
		SettingsRow,
		SettingsSection,
		Switch,
		icons,
		toast
	} from '@facile/muse';
	import { backend, type Webhook } from '$lib/backend';
	import WebhookDrawer from './WebhookDrawer.svelte';

	let webhooks = $state<Webhook[]>([]);
	let loading = $state(false);
	let drawerOpen = $state(false);
	let editing = $state<Webhook | null>(null);
	let deleting = $state<Webhook | null>(null);
	let testingId = $state<number | null>(null);

	onMount(load);

	async function load() {
		loading = true;
		try {
			webhooks = await backend.listWebhooks();
		} catch (err) {
			toast.danger(err instanceof Error ? err.message : 'Could not load webhooks.');
		} finally {
			loading = false;
		}
	}

	function openCreate() {
		editing = null;
		drawerOpen = true;
	}

	function openEdit(webhook: Webhook) {
		editing = webhook;
		drawerOpen = true;
	}

	async function toggle(webhook: Webhook) {
		const next = !webhook.enabled;
		try {
			const updated = await backend.updateWebhook(webhook.id, {
				url: webhook.url,
				name: webhook.name,
				secret_header: webhook.secret_header,
				enabled: next
			});
			webhooks = webhooks.map((w) => (w.id === updated.id ? updated : w));
			toast.success(next ? 'Webhook enabled.' : 'Webhook disabled.');
		} catch (err) {
			toast.danger(err instanceof Error ? err.message : 'Could not update webhook.');
		}
	}

	async function test(webhook: Webhook) {
		testingId = webhook.id;
		try {
			const res = await backend.testWebhook(webhook.id);
			if (res.ok) {
				toast.success(`HTTP ${res.status_code} in ${res.latency_ms}ms`);
			} else {
				toast.danger(res.error || `HTTP ${res.status_code}`);
			}
		} catch (err) {
			toast.danger(err instanceof Error ? err.message : 'Test failed.');
		} finally {
			testingId = null;
		}
	}

	function handleSaved(saved: Webhook) {
		const idx = webhooks.findIndex((w) => w.id === saved.id);
		if (idx >= 0) {
			webhooks[idx] = saved;
		} else {
			webhooks = [...webhooks, saved];
		}
	}

	async function confirmDelete() {
		if (!deleting) return;
		const id = deleting.id;
		try {
			await backend.deleteWebhook(id);
			webhooks = webhooks.filter((w) => w.id !== id);
			toast.success('Webhook deleted.');
		} catch (err) {
			toast.danger(err instanceof Error ? err.message : 'Could not delete webhook.');
		} finally {
			deleting = null;
		}
	}
</script>

<SettingsSection
	title="Webhooks"
	description="HTTP POST endpoints notified immediately when a monitor goes down or recovers."
>
	{#if webhooks.length === 0 && !loading}
		<div class="rounded-fc-lg border border-fc-border bg-fc-surface-subtle p-fc-6 text-center">
			<p class="text-fc-sm text-fc-fg-muted mb-fc-4">
				No webhooks configured. Add an endpoint (like Antenne) to receive instant notifications.
			</p>
			<Button icon={icons.plus} onclick={openCreate}>Add webhook</Button>
		</div>
	{:else}
		<div class="flex justify-end mb-fc-2">
			<Button icon={icons.plus} variant="outline" onclick={openCreate}>Add webhook</Button>
		</div>

		<div class="flex flex-col gap-fc-3">
			{#each webhooks as webhook (webhook.id)}
				<div class="rounded-fc-lg border border-fc-border bg-fc-surface p-fc-4 flex items-center justify-between gap-fc-4">
					<div class="flex flex-col gap-fc-1 min-w-0">
						<div class="flex items-center gap-fc-2">
							<span class="font-medium text-fc-fg text-fc-sm">
								{webhook.name || 'Webhook'}
							</span>
							{#if !webhook.enabled}
								<span class="rounded bg-fc-neutral-subtle px-1.5 py-0.5 text-fc-xs text-fc-fg-muted font-medium">
									Disabled
								</span>
							{/if}
						</div>
						<div class="font-fc-mono text-fc-xs text-fc-fg-muted truncate max-w-md">
							{webhook.url}
						</div>
					</div>

					<div class="flex items-center gap-fc-2 shrink-0">
						<Button
							variant="outline"
							size="sm"
							icon={icons.refresh}
							disabled={testingId === webhook.id}
							onclick={() => test(webhook)}
						>
							{testingId === webhook.id ? 'Testing…' : 'Test'}
						</Button>
						<IconButton
							variant="ghost"
							aria-label="Edit webhook"
							onclick={() => openEdit(webhook)}
						>
							<Icon icon={icons.edit} size={18} />
						</IconButton>
						<IconButton
							variant="danger"
							aria-label="Delete webhook"
							onclick={() => (deleting = webhook)}
						>
							<Icon icon={icons.remove} size={18} />
						</IconButton>
						<Switch checked={webhook.enabled} onchange={() => toggle(webhook)} />
					</div>
				</div>
			{/each}
		</div>
	{/if}
</SettingsSection>

<WebhookDrawer
	bind:open={drawerOpen}
	editing={editing}
	onSaved={handleSaved}
/>

<ConfirmModal
	open={Boolean(deleting)}
	tone="danger"
	title="Delete webhook?"
	description="This endpoint will no longer receive alerts when monitors transition."
	confirmLabel="Delete webhook"
	onConfirm={confirmDelete}
	onCancel={() => (deleting = null)}
/>
