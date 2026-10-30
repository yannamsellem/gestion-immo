<script lang="ts">
	import { enhance } from '$app/forms';
	import { invalidateAll } from '$app/navigation';
	import { resolve } from '$app/paths';
	import IconDownload from '@tabler/icons-svelte/icons/download';
	import IconFileUpload from '@tabler/icons-svelte/icons/file-upload';
	import { toast } from 'svelte-sonner';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import * as Select from '#lib/components/ui/select/index.js';
	import { Separator } from '#lib/components/ui/separator/index.js';
	import * as Sheet from '#lib/components/ui/sheet/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import { formatIsoDateFr } from '#lib/core/formatters/date';
	import type { DocumentTemplateDTO, StoredFileDTO } from '#lib/modules/document/types';
	import type { LeaseDTO } from '#lib/modules/lease/types';

	let {
		open = $bindable(false),
		selectedLease,
		entityName,
		tenantName,
		propertyLotLabel,
		templates,
		documents
	}: {
		open?: boolean;
		selectedLease: LeaseDTO | null;
		entityName: string;
		tenantName: string;
		propertyLotLabel: string;
		templates: DocumentTemplateDTO[];
		documents: StoredFileDTO[];
	} = $props();

	let selectedTemplateId = $state('');
	let uploadCategory = $state('BAIL_SIGNE');
	let hasUploadFile = $state(false);
	let isGeneratingDoc = $state(false);
	let isUploadingDoc = $state(false);

	$effect(() => {
		if (templates.length > 0 && !templates.some((tpl) => tpl.id === selectedTemplateId)) {
			selectedTemplateId = templates[0].id;
		}
	});
</script>

<Sheet.Root bind:open>
	<Sheet.Content side="right" class="w-full overflow-y-auto sm:max-w-lg">
		<Sheet.Header class="border-b pb-4">
			<Sheet.Title>Contrat & Coffre-fort documentaire R2</Sheet.Title>
			<Sheet.Description>
				Générez un document à partir d’un modèle ou téléversez une pièce justificative (bail signé,
				attestation d’assurance, RIB).
			</Sheet.Description>
		</Sheet.Header>

		{#if selectedLease}
			<div class="flex flex-col gap-6 p-4">
				<!-- 1. Generate from template -->
				<div class="flex flex-col gap-3">
					<h3 class="text-xs font-semibold uppercase">1. Générer depuis un modèle</h3>
					{#if templates.length > 0}
						<form
							method="POST"
							action="?/generateDoc"
							use:enhance={() => {
								isGeneratingDoc = true;
								return async ({ result }) => {
									isGeneratingDoc = false;
									if (result.type === 'success') {
										const payload = result.data as { successMessage?: string } | undefined;
										toast.success(payload?.successMessage ?? 'Document généré.');
										await invalidateAll();
										return;
									}
									if (result.type === 'failure') {
										const payload = result.data as { actionError?: string } | undefined;
										toast.error(payload?.actionError ?? 'Échec de la génération.');
										return;
									}
									if (result.type === 'error') {
										toast.error(result.error?.message ?? 'Erreur serveur lors de la génération.');
									}
								};
							}}
							class="flex gap-2"
						>
							<input type="hidden" name="lease_id" value={selectedLease.id} />
							<input type="hidden" name="legal_entity_id" value={selectedLease.legal_entity_id} />
							<input type="hidden" name="template_id" value={selectedTemplateId} />
							<input type="hidden" name="tenant_name" value={tenantName} />
							<input type="hidden" name="entity_name" value={entityName} />
							<input type="hidden" name="property_label" value={propertyLotLabel} />
							<input type="hidden" name="rent_net_cents" value={selectedLease.rent_net_cents} />
							<input type="hidden" name="charges_cents" value={selectedLease.charges_cents} />

							<div class="flex-1">
								<Select.Root
									type="single"
									value={selectedTemplateId}
									onValueChange={(next) => {
										if (next) selectedTemplateId = next;
									}}
								>
									<Select.Trigger class="w-full">
										{templates.find((tpl) => tpl.id === selectedTemplateId)?.name ??
											'Sélectionner un modèle'}
									</Select.Trigger>
									<Select.Content>
										<Select.Group>
											{#each templates as template (template.id)}
												<Select.Item value={template.id} label={template.name}>
													{template.name} ({template.format})
												</Select.Item>
											{/each}
										</Select.Group>
									</Select.Content>
								</Select.Root>
							</div>
							<Button
								type="submit"
								variant="secondary"
								disabled={isGeneratingDoc || !selectedTemplateId}
							>
								{#if isGeneratingDoc}
									<Spinner data-icon="inline-start" />
								{/if}
								Générer
							</Button>
						</form>
					{:else}
						<p class="text-xs text-muted-foreground">
							Aucun modèle enregistré. Ajoutez-en un dans
							<a
								href={resolve('/(app)/leases/templates')}
								class="font-medium underline underline-offset-4"
							>
								Modèles de documents
							</a>.
						</p>
					{/if}
				</div>

				<Separator />

				<!-- 2. Upload file to R2 -->
				<form
					method="POST"
					action="?/uploadFile"
					enctype="multipart/form-data"
					use:enhance={({ formElement }) => {
						isUploadingDoc = true;
						return async ({ result }) => {
							isUploadingDoc = false;
							if (result.type === 'success') {
								const payload = result.data as { successMessage?: string } | undefined;
								toast.success(
									payload?.successMessage ??
										'Pièce justificative téléversée dans le coffre-fort R2.'
								);
								hasUploadFile = false;
								formElement.reset();
								uploadCategory = 'BAIL_SIGNE';
								await invalidateAll();
								return;
							}
							if (result.type === 'failure') {
								const payload = result.data as { actionError?: string } | undefined;
								toast.error(payload?.actionError ?? 'Échec du téléversement.');
								return;
							}
							if (result.type === 'error') {
								toast.error(result.error?.message ?? 'Erreur serveur lors du téléversement.');
							}
						};
					}}
					class="flex flex-col gap-3"
				>
					<input type="hidden" name="lease_id" value={selectedLease.id} />
					<input type="hidden" name="legal_entity_id" value={selectedLease.legal_entity_id} />

					<h3 class="text-xs font-semibold uppercase">2. Téléverser une pièce (Cloudflare R2)</h3>
					<Field.Group>
						<Field.Field>
							<Field.Label for="lease-upload-category">Catégorie de pièce</Field.Label>
							<Input
								id="lease-upload-category"
								name="category"
								placeholder="BAIL_SIGNE, ATTESTATION_ASSURANCE, CAUTION..."
								bind:value={uploadCategory}
								required
							/>
						</Field.Field>

						<Field.Field>
							<Field.Label for="lease-upload-file">Fichier PDF / DOCX / Image</Field.Label>
							<Input
								id="lease-upload-file"
								name="file"
								type="file"
								onchange={(event) => {
									hasUploadFile = Boolean((event.currentTarget as HTMLInputElement).files?.length);
								}}
								required
							/>
						</Field.Field>
					</Field.Group>

					<Button type="submit" variant="outline" disabled={isUploadingDoc || !hasUploadFile}>
						{#if isUploadingDoc}
							<Spinner data-icon="inline-start" />
						{:else}
							<IconFileUpload data-icon="inline-start" />
						{/if}
						Téléverser dans le dossier du bail
					</Button>
				</form>

				<Separator />

				<!-- 3. Existing archived documents -->
				<div class="flex flex-col gap-2">
					<h3 class="text-xs font-semibold uppercase">
						3. Pièces archivées ({documents.length})
					</h3>
					{#if documents.length === 0}
						<p class="text-xs text-muted-foreground">Aucune pièce archivée sur ce bail.</p>
					{:else}
						{#each documents as doc (doc.id)}
							<div class="flex items-center justify-between rounded-md border p-2.5 text-xs">
								<div class="flex min-w-0 flex-col">
									<span class="truncate font-medium">{doc.file_name}</span>
									<span class="text-[11px] text-muted-foreground">
										{doc.category} · {formatIsoDateFr(doc.created_at)}
									</span>
								</div>
								{#if doc.download_url}
									<Button
										size="sm"
										variant="outline"
										href={doc.download_url}
										target="_blank"
										rel="noreferrer"
									>
										<IconDownload data-icon="inline-start" />
										Télécharger
									</Button>
								{/if}
							</div>
						{/each}
					{/if}
				</div>
			</div>
		{/if}
	</Sheet.Content>
</Sheet.Root>
