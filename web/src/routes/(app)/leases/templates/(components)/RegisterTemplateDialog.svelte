<script lang="ts">
	import { enhance } from '$app/forms';
	import { invalidateAll } from '$app/navigation';
	import IconAlertTriangle from '@tabler/icons-svelte/icons/alert-triangle';
	import { toast } from 'svelte-sonner';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Dialog from '#lib/components/ui/dialog/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import * as Select from '#lib/components/ui/select/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import {
		formatTemplateTypeLabel,
		TEMPLATE_FORMAT_OPTIONS,
		TEMPLATE_TYPE_OPTIONS
	} from '#lib/modules/document/labels';
	import type {
		DocumentTemplateDTO,
		TemplateFormat,
		TemplateType
	} from '#lib/modules/document/types';

	const DEFAULT_HTML_BODY =
		'<h1>{{entity_name}}</h1><p>Locataire : {{tenant_name}}</p><p>Lot : {{property_label}}</p>';

	let {
		open = $bindable(false),
		onRegistered
	}: {
		open?: boolean;
		onRegistered?: (template: DocumentTemplateDTO) => void;
	} = $props();

	let name = $state('');
	let templateType = $state<TemplateType>('LEASE');
	let format = $state<TemplateFormat>('HTML');
	let htmlBody = $state(DEFAULT_HTML_BODY);
	let docxBase64 = $state('');
	let isSubmitting = $state(false);
	let errorMessage = $state<string | null>(null);

	function resetForm(): void {
		name = '';
		templateType = 'LEASE';
		format = 'HTML';
		htmlBody = DEFAULT_HTML_BODY;
		docxBase64 = '';
		errorMessage = null;
	}

	async function handleDocxFileInput(event: Event): Promise<void> {
		const input = event.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		if (!file) {
			docxBase64 = '';
			return;
		}
		const buffer = await file.arrayBuffer();
		const bytes = new Uint8Array(buffer);
		let binary = '';
		for (const byte of bytes) {
			binary += String.fromCharCode(byte);
		}
		docxBase64 = btoa(binary);
	}
</script>

<Dialog.Root
	bind:open
	onOpenChange={(next) => {
		if (!next) {
			errorMessage = null;
		}
	}}
>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>Enregistrer un modèle de document</Dialog.Title>
			<Dialog.Description>
				Supporte les modèles Word <code>.docx</code> avec balises
				<code>&#123;&#123;variable&#125;&#125;</code>
				et les gabarits HTML convertis en PDF.
			</Dialog.Description>
		</Dialog.Header>

		{#if errorMessage}
			<Alert.Root variant="destructive">
				<IconAlertTriangle />
				<Alert.Title>Enregistrement impossible</Alert.Title>
				<Alert.Description>{errorMessage}</Alert.Description>
			</Alert.Root>
		{/if}

		<form
			method="POST"
			action="?/registerTemplate"
			use:enhance={() => {
				isSubmitting = true;
				errorMessage = null;
				return async ({ result }) => {
					isSubmitting = false;
					if (result.type === 'success') {
						const payload = result.data as
							{ registeredTemplate?: DocumentTemplateDTO; successMessage?: string } | undefined;
						toast.success(payload?.successMessage ?? 'Modèle documentaire enregistré.');
						resetForm();
						open = false;
						await invalidateAll();
						if (payload?.registeredTemplate) {
							onRegistered?.(payload.registeredTemplate);
						}
						return;
					}
					if (result.type === 'failure') {
						const payload = result.data as { actionError?: string } | undefined;
						const message = payload?.actionError ?? 'Impossible d’enregistrer le modèle.';
						errorMessage = message;
						toast.error(message);
						return;
					}
					if (result.type === 'error') {
						const message =
							result.error?.message ?? 'Erreur serveur lors de l’enregistrement du modèle.';
						errorMessage = message;
						toast.error(message);
					}
				};
			}}
			class="flex flex-col gap-4"
		>
			<input type="hidden" name="template_type" value={templateType} />
			<input type="hidden" name="format" value={format} />
			<input type="hidden" name="docx_base64" value={docxBase64} />

			<Field.Group>
				<Field.Field>
					<Field.Label for="register-template-name">Nom du modèle</Field.Label>
					<Input
						id="register-template-name"
						name="name"
						placeholder="Bail Habitation Loi 89 Standard"
						bind:value={name}
						required
					/>
				</Field.Field>

				<div class="grid gap-3 sm:grid-cols-2">
					<Field.Field>
						<Field.Label>Catégorie</Field.Label>
						<Select.Root
							type="single"
							value={templateType}
							onValueChange={(next) => {
								if (next) templateType = next;
							}}
						>
							<Select.Trigger class="w-full">
								{formatTemplateTypeLabel(templateType)}
							</Select.Trigger>
							<Select.Content>
								<Select.Group>
									{#each TEMPLATE_TYPE_OPTIONS as option (option.value)}
										<Select.Item value={option.value} label={option.label}>
											{option.label}
										</Select.Item>
									{/each}
								</Select.Group>
							</Select.Content>
						</Select.Root>
					</Field.Field>

					<Field.Field>
						<Field.Label>Format</Field.Label>
						<Select.Root
							type="single"
							value={format}
							onValueChange={(next) => {
								if (next) format = next;
							}}
						>
							<Select.Trigger class="w-full">
								{TEMPLATE_FORMAT_OPTIONS.find((opt) => opt.value === format)?.label ?? format}
							</Select.Trigger>
							<Select.Content>
								<Select.Group>
									{#each TEMPLATE_FORMAT_OPTIONS as option (option.value)}
										<Select.Item value={option.value} label={option.label}>
											{option.label}
										</Select.Item>
									{/each}
								</Select.Group>
							</Select.Content>
						</Select.Root>
					</Field.Field>
				</div>

				{#if format === 'HTML'}
					<Field.Field>
						<Field.Label for="register-template-html">Gabarit HTML</Field.Label>
						<textarea
							id="register-template-html"
							name="html_body"
							rows="4"
							class="w-full rounded-md border border-input bg-background p-2 font-mono text-xs"
							bind:value={htmlBody}
							required></textarea>
					</Field.Field>
				{:else}
					<Field.Field>
						<Field.Label for="register-template-docx">Fichier .docx source</Field.Label>
						<Input
							id="register-template-docx"
							type="file"
							accept=".docx"
							onchange={handleDocxFileInput}
							required
						/>
					</Field.Field>
				{/if}
			</Field.Group>

			<Dialog.Footer>
				<Button
					variant="outline"
					type="button"
					disabled={isSubmitting}
					onclick={() => {
						open = false;
					}}
				>
					Annuler
				</Button>
				<Button type="submit" disabled={isSubmitting}>
					{#if isSubmitting}
						<Spinner data-icon="inline-start" />
					{/if}
					Enregistrer le modèle
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
