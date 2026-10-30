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
	import type { LegalEntityDTO } from '#lib/modules/portfolio/types';

	let {
		open = $bindable(false),
		entities,
		defaultEntityId = ''
	}: {
		open?: boolean;
		entities: LegalEntityDTO[];
		defaultEntityId?: string;
	} = $props();

	let entityId = $state('');
	let csvContent = $state('');
	let isSubmitting = $state(false);
	let errorMessage = $state<string | null>(null);

	$effect(() => {
		if (open) {
			const preferred =
				defaultEntityId && entities.some((entity) => entity.id === defaultEntityId)
					? defaultEntityId
					: (entities[0]?.id ?? '');
			if (preferred) {
				entityId = preferred;
			}
		}
	});

	function resetForm(): void {
		csvContent = '';
		errorMessage = null;
	}

	async function handleCsvFileChange(event: Event): Promise<void> {
		const input = event.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		if (!file) return;
		csvContent = await file.text();
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
			<Dialog.Title>Importer un relevé bancaire CSV</Dialog.Title>
			<Dialog.Description>
				Importe et impute automatiquement les virements de loyers (colonnes :
				<code>lease_id,amount_cents,payment_date,label,external_ref</code>).
			</Dialog.Description>
		</Dialog.Header>

		{#if errorMessage}
			<Alert.Root variant="destructive">
				<IconAlertTriangle />
				<Alert.Title>Import impossible</Alert.Title>
				<Alert.Description>{errorMessage}</Alert.Description>
			</Alert.Root>
		{/if}

		<form
			method="POST"
			action="?/importCsv"
			use:enhance={({ formElement }) => {
				isSubmitting = true;
				errorMessage = null;
				return async ({ result }) => {
					isSubmitting = false;
					if (result.type === 'success') {
						const payload = result.data as { successMessage?: string } | undefined;
						toast.success(payload?.successMessage ?? 'Relevé CSV bancaire importé.');
						resetForm();
						formElement.reset();
						open = false;
						await invalidateAll();
						return;
					}
					if (result.type === 'failure') {
						const payload = result.data as { actionError?: string } | undefined;
						const message = payload?.actionError ?? 'Échec de l’import bancaire CSV.';
						errorMessage = message;
						toast.error(message);
						return;
					}
					if (result.type === 'error') {
						const message = result.error?.message ?? 'Erreur serveur lors de l’import CSV.';
						errorMessage = message;
						toast.error(message);
					}
				};
			}}
			class="flex flex-col gap-4"
		>
			<input type="hidden" name="entity_id" value={entityId} />

			<Field.Group>
				<Field.Field>
					<Field.Label>Entité juridique</Field.Label>
					<Select.Root
						type="single"
						value={entityId}
						onValueChange={(next) => {
							if (next) entityId = next;
						}}
					>
						<Select.Trigger class="w-full">
							{entities.find((entity) => entity.id === entityId)?.name ?? 'Sélectionner une entité'}
						</Select.Trigger>
						<Select.Content>
							<Select.Group>
								{#each entities as entity (entity.id)}
									<Select.Item value={entity.id} label={entity.name}>{entity.name}</Select.Item>
								{/each}
							</Select.Group>
						</Select.Content>
					</Select.Root>
				</Field.Field>

				<Field.Field>
					<Field.Label for="import-csv-file">Fichier CSV</Field.Label>
					<Input
						id="import-csv-file"
						type="file"
						accept=".csv,text/csv"
						onchange={handleCsvFileChange}
					/>
				</Field.Field>

				<Field.Field>
					<Field.Label for="import-csv-raw">Contenu CSV brut</Field.Label>
					<textarea
						id="import-csv-raw"
						name="csv_content"
						rows="4"
						class="w-full rounded-md border border-input bg-background p-2 font-mono text-xs placeholder:text-muted-foreground"
						placeholder="lease_id,amount_cents,payment_date,label,external_ref"
						bind:value={csvContent}
						required></textarea>
				</Field.Field>
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
				<Button type="submit" disabled={isSubmitting || !entityId}>
					{#if isSubmitting}
						<Spinner data-icon="inline-start" />
					{/if}
					Importer et imputer
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
