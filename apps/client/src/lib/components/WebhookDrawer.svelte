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
			toast.danger('Enter a webhook URL first.');
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
				toast.success(`Endpoint responded with HTTP ${res.status_code} in ${res.latency_ms}ms`);
			} else {
				toast.danger(res.error || `Endpoint returned HTTP ${res.status_code}`);
			}
		} catch (err) {
			toast.danger(err instanceof Error ? err.message : 'Could not test the webhook.');
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
			toast.success(editing ? 'Webhook updated.' : 'Webhook added.');
		} catch (err) {
			error = err instanceof Error ? err.message : 'Could not save the webhook.';
			toast.danger(error);
		} finally {
			submitting = false;
		}
	}
</script>

<Drawer bind:open title={editing ? 'Edit webhook' : 'Add webhook'}>
	<form onsubmit={save} class="flex flex-col gap-4">
		{#if error}
			<Alert tone="danger" title={error} />
		{/if}

		<Field for="webhook-name" label="Name">
			<Input id="webhook-name" bind:value={name} placeholder="e.g. Antenne Alert Hub" />
		</Field>

		<Field for="webhook-url" label="Webhook URL">
			<Input id="webhook-url" bind:value={url} placeholder="https://antenne.facile.studio/webhook/sonde" required />
			<div class="mt-1 flex items-center justify-between text-fc-xs text-fc-fg-muted">
				<span>HTTP POST recipient for monitor.down and monitor.up alerts.</span>
				<button
					type="button"
					onclick={fillAntenne}
					class="text-fc-accent hover:underline font-medium cursor-pointer"
				>
					Use Antenne preset
				</button>
			</div>
		</Field>

		<Field for="webhook-secret" label="HMAC secret">
			<SecretField id="webhook-secret" bind:value={secret} placeholder="Optional shared signing key" />
		</Field>

		<Field for="webhook-header" label="Signature header">
			<Input id="webhook-header" bind:value={secretHeader} placeholder="x-sonde-signature-256" />
		</Field>

		<div class="flex items-center justify-between py-2">
			<div>
				<div class="text-fc-sm font-medium text-fc-fg">Enable webhook</div>
				<div class="text-fc-xs text-fc-fg-muted">Send notifications on monitor state changes.</div>
			</div>
			<Switch bind:checked={enabled} />
		</div>

		<div class="mt-4 flex items-center justify-between gap-2">
			<Button
				type="button"
				variant="outline"
				icon={icons.refresh}
				disabled={!ready || testing}
				onclick={testEndpoint}
			>
				{testing ? 'Testing…' : 'Test endpoint'}
			</Button>

			<div class="flex items-center gap-2">
				<Button type="button" variant="outline" onclick={() => (open = false)}>Cancel</Button>
				<Button type="submit" disabled={!ready || submitting} icon={icons.check}>
					{submitting ? 'Saving…' : editing ? 'Save changes' : 'Add webhook'}
				</Button>
			</div>
		</div>
	</form>
</Drawer>
