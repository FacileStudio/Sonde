<script lang="ts">
	import {
		Alert,
		Button,
		Drawer,
		Field,
		Input,
		SecretField,
		Switch,
		icons,
		toast
	} from '@facile/muse';
	import { backend, type Webhook, type WebhookInput } from '$lib/backend';
	import { milliseconds } from '$lib/format';
	import { t } from '$lib/i18n/index.svelte';

	let {
		open = $bindable(false),
		editing = null,
		onSaved
	}: {
		open?: boolean;
		editing?: Webhook | null;
		onSaved: (webhook: Webhook) => void;
	} = $props();

	let name = $state('');
	let url = $state('');
	let secret = $state('');
	let secretHeader = $state('x-sonde-signature-256');
	let enabled = $state(true);
	let error = $state('');
	let submitting = $state(false);
	let testing = $state(false);

	const ready = $derived(Boolean(url.trim()));

	$effect(() => {
		if (!open) {
			name = '';
			url = '';
			secret = '';
			secretHeader = 'x-sonde-signature-256';
			enabled = true;
			error = '';
			return;
		}
		if (editing) {
			name = editing.name;
			url = editing.url;
			secret = editing.secret ?? '';
			secretHeader = editing.secret_header || 'x-sonde-signature-256';
			enabled = editing.enabled;
		}
	});

	function fillAntenne() {
		if (!name.trim()) name = 'Antenne';
		url = 'https://antenne.facile.studio/webhook/sonde';
		secretHeader = 'x-sonde-signature-256';
	}

	async function testEndpoint() {
		if (!url.trim()) {
			toast.danger(t('webhookDrawer.urlFirst'));
			return;
		}
		testing = true;
		try {
			const res = await backend.testWebhook({
				name: name.trim(),
				url: url.trim(),
				secret: secret.trim(),
				secret_header: secretHeader.trim(),
				enabled
			});
			if (res.ok) {
				toast.success(
					t('webhookDrawer.responded', { status: res.status_code, latency: milliseconds(res.latency_ms) })
				);
			} else {
				toast.danger(res.error || t('webhookDrawer.returned', { status: res.status_code }));
			}
		} catch (err) {
			toast.danger(err instanceof Error ? err.message : t('webhooks.error.test'));
		} finally {
			testing = false;
		}
	}

	async function save(event: SubmitEvent) {
		event.preventDefault();
		if (submitting || !ready) return;
		error = '';
		submitting = true;
		const input: WebhookInput = {
			name: name.trim(),
			url: url.trim(),
			secret: secret.trim(),
			secret_header: secretHeader.trim() || 'x-sonde-signature-256',
			enabled
		};
		try {
			const saved = editing
				? await backend.updateWebhook(editing.id, input)
				: await backend.createWebhook(input);
			open = false;
			onSaved(saved);
			toast.success(editing ? t('webhooks.toast.updated') : t('webhooks.toast.added'));
		} catch (err) {
			error = err instanceof Error ? err.message : t('webhooks.error.save');
			toast.danger(error);
		} finally {
			submitting = false;
		}
	}
</script>

<Drawer bind:open title={editing ? t('webhookDrawer.edit') : t('webhooks.add')}>
	<form onsubmit={save} class="flex flex-col gap-4">
		{#if error}
			<Alert tone="danger" title={error} />
		{/if}

		<Field for="webhook-name" label={t('common.name')}>
			<Input id="webhook-name" bind:value={name} placeholder={t('webhookDrawer.namePlaceholder')} />
		</Field>

		<Field for="webhook-url" label={t('webhookDrawer.url')}>
			<Input id="webhook-url" bind:value={url} placeholder="https://antenne.facile.studio/webhook/sonde" required />
			<div class="mt-1 flex items-center justify-between text-fc-xs text-fc-fg-muted">
				<span>{t('webhookDrawer.urlHint')}</span>
				<button
					type="button"
					onclick={fillAntenne}
					class="text-fc-accent hover:underline font-medium cursor-pointer"
				>
					{t('webhookDrawer.preset')}
				</button>
			</div>
		</Field>

		<Field for="webhook-secret" label={t('webhookDrawer.secret')}>
			<SecretField
				id="webhook-secret"
				editable
				copyable={false}
				bind:value={secret}
				placeholder={t('webhookDrawer.secretPlaceholder')}
			/>
		</Field>

		<Field for="webhook-header" label={t('webhookDrawer.header')}>
			<Input id="webhook-header" bind:value={secretHeader} placeholder="x-sonde-signature-256" />
		</Field>

		<div class="flex items-center justify-between py-2">
			<div>
				<div class="text-fc-sm font-medium text-fc-fg">{t('webhookDrawer.enable')}</div>
				<div class="text-fc-xs text-fc-fg-muted">{t('webhookDrawer.enableHint')}</div>
			</div>
			<Switch bind:checked={enabled} />
		</div>

		<div class="mt-4 flex items-center justify-between gap-2">
			<Button
				type="button"
				variant="secondary"
				icon={icons.refresh}
				disabled={!ready || testing}
				onclick={testEndpoint}
			>
				{testing ? t('webhooks.testing') : t('webhookDrawer.testEndpoint')}
			</Button>

			<div class="flex items-center gap-2">
				<Button type="button" variant="secondary" onclick={() => (open = false)}>{t('common.cancel')}</Button>
				<Button type="submit" disabled={!ready || submitting} icon={icons.check}>
					{submitting ? t('common.saving') : editing ? t('common.saveChanges') : t('webhooks.add')}
				</Button>
			</div>
		</div>
	</form>
</Drawer>
