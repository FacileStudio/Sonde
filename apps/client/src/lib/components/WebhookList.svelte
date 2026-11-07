<script lang="ts">
	import { onMount } from 'svelte';
	import {
		Badge,
		Button,
		ConfirmModal,
		EmptyState,
		SettingsSection,
		Skeleton,
		StatusDot,
		Switch,
		Table,
		icons,
		toast
	} from '@facile/muse';
	import { backend, type Webhook } from '$lib/backend';
	import { milliseconds } from '$lib/format';
	import { t } from '$lib/i18n/index.svelte';
	import WebhookDrawer from './WebhookDrawer.svelte';

	let webhooks = $state<Webhook[]>([]);
	let loading = $state(true);
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
			toast.danger(err instanceof Error ? err.message : t('webhooks.error.load'));
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
			toast.success(next ? t('webhooks.toast.enabled') : t('webhooks.toast.disabled'));
		} catch (err) {
			toast.danger(err instanceof Error ? err.message : t('webhooks.error.update'));
		}
	}

	async function test(webhook: Webhook) {
		testingId = webhook.id;
		try {
			const res = await backend.testWebhook(webhook.id);
			if (res.ok) {
				toast.success(
					t('webhooks.testResult', { status: res.status_code, latency: milliseconds(res.latency_ms) })
				);
			} else {
				toast.danger(res.error || t('webhooks.testStatus', { status: res.status_code }));
			}
		} catch (err) {
			toast.danger(err instanceof Error ? err.message : t('webhooks.error.testFailed'));
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
			toast.success(t('webhooks.toast.deleted'));
		} catch (err) {
			toast.danger(err instanceof Error ? err.message : t('webhooks.error.delete'));
		} finally {
			deleting = null;
		}
	}

	const host = (url: string) => {
		try {
			return new URL(url).host;
		} catch {
			return url;
		}
	};
</script>

<!-- `bare` because the body is a Table, which brings its own surface. A Table
     inside SettingsSection's default Card is a container inside a container. -->
<SettingsSection
	title={t('webhooks.title')}
	description={t('webhooks.description')}
	bare
>
	{#snippet actions()}
		<Button size="sm" icon={icons.plus} onclick={openCreate}>{t('webhooks.add')}</Button>
	{/snippet}

	{#if loading}
		<Skeleton class="h-24 w-full" />
	{:else if webhooks.length === 0}
		<EmptyState
			icon={icons.plug}
			title={t('webhooks.empty.title')}
			description={t('webhooks.empty.description')}
		>
			<Button variant="secondary" icon={icons.plus} onclick={openCreate}>{t('webhooks.empty.cta')}</Button>
		</EmptyState>
	{:else}
		<Table>
			<thead>
				<tr>
					<th scope="col">{t('webhooks.col.endpoint')}</th>
					<th scope="col">{t('webhooks.col.signature')}</th>
					<th scope="col">{t('webhooks.col.state')}</th>
					<th scope="col"><span class="sr-only">{t('common.actions')}</span></th>
				</tr>
			</thead>
			<tbody>
				{#each webhooks as webhook (webhook.id)}
					<tr class={webhook.enabled ? '' : 'opacity-55'}>
						<td>
							<div class="flex min-w-0 flex-col gap-1">
								<span class="truncate font-medium text-fc-fg">{webhook.name || host(webhook.url)}</span>
								<span class="truncate font-fc-mono text-fc-xs text-fc-fg-muted">{webhook.url}</span>
							</div>
						</td>
						<td>
							{#if webhook.secret_header}
								<span class="font-fc-mono text-fc-xs text-fc-fg-muted">{webhook.secret_header}</span>
							{:else}
								<Badge tone="warning">{t('webhooks.unsigned')}</Badge>
							{/if}
						</td>
						<td>
							<StatusDot
								tone={webhook.enabled ? 'success' : 'neutral'}
								label={webhook.enabled ? t('common.enabled') : t('common.disabled')}
							/>
						</td>
						<td>
							<div class="flex items-center justify-end gap-2">
								<Button
									variant="secondary"
									size="sm"
									icon={icons.refresh}
									disabled={testingId === webhook.id}
									onclick={() => test(webhook)}
								>
									{testingId === webhook.id ? t('webhooks.testing') : t('webhooks.test')}
								</Button>
								<Button
									variant="ghost"
									size="sm"
									icon={icons.edit}
									aria-label={t('webhooks.editAria', { name: webhook.name || webhook.url })}
									onclick={() => openEdit(webhook)}
								/>
								<Button
									variant="ghost-danger"
									size="sm"
									icon={icons.remove}
									aria-label={t('webhooks.deleteAria', { name: webhook.name || webhook.url })}
									onclick={() => (deleting = webhook)}
								/>
								<Switch
									checked={webhook.enabled}
									onchange={() => toggle(webhook)}
									aria-label={t('webhooks.enableAria', { name: webhook.name || webhook.url })}
								/>
							</div>
						</td>
					</tr>
				{/each}
			</tbody>
		</Table>
	{/if}
</SettingsSection>

<WebhookDrawer bind:open={drawerOpen} {editing} onSaved={handleSaved} />

<ConfirmModal
	open={Boolean(deleting)}
	tone="danger"
	title={t('webhooks.delete.title')}
	description={t('webhooks.delete.description')}
	confirmLabel={t('webhooks.delete.confirm')}
	onConfirm={confirmDelete}
	onCancel={() => (deleting = null)}
/>
