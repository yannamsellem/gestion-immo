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
	import { formatLotTypeLabel, LOT_TYPE_OPTIONS } from '#lib/modules/portfolio/labels';
	import type { PropertyDTO } from '#lib/modules/portfolio/types';

	let {
		open = $bindable(false),
		properties,
		defaultPropertyId = ''
	}: {
		open?: boolean;
		properties: PropertyDTO[];
		defaultPropertyId?: string;
	} = $props();

	let propertyId = $state('');
	let label = $state('');
	let lotType = $state('APARTMENT');
	let surfaceM2 = $state('45');
	let ownershipShare = $state('250');
	let floor = $state('1er étage');
	let isSubmitting = $state(false);
	let errorMessage = $state<string | null>(null);

	$effect(() => {
		if (open) {
			const preferred =
				defaultPropertyId && properties.some((property) => property.id === defaultPropertyId)
					? defaultPropertyId
					: (properties[0]?.id ?? '');
			if (preferred) {
				propertyId = preferred;
			}
		}
	});

	function resetForm(): void {
		label = '';
		lotType = 'APARTMENT';
		surfaceM2 = '45';
		ownershipShare = '250';
		floor = '1er étage';
		errorMessage = null;
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
			<Dialog.Title>Ajouter un lot locatif</Dialog.Title>
			<Dialog.Description>
				Créez une unité locative rattachée à un bien immobilier existant.
			</Dialog.Description>
		</Dialog.Header>

		{#if errorMessage}
			<Alert.Root variant="destructive">
				<IconAlertTriangle />
				<Alert.Title>Création impossible</Alert.Title>
				<Alert.Description>{errorMessage}</Alert.Description>
			</Alert.Root>
		{/if}

		<form
			method="POST"
			action="?/createLot"
			use:enhance={() => {
				isSubmitting = true;
				errorMessage = null;
				return async ({ result }) => {
					isSubmitting = false;
					if (result.type === 'success') {
						const payload = result.data as { successMessage?: string } | undefined;
						toast.success(payload?.successMessage ?? 'Lot ajouté.');
						resetForm();
						open = false;
						await invalidateAll();
						return;
					}
					if (result.type === 'failure') {
						const payload = result.data as { actionError?: string } | undefined;
						const message = payload?.actionError ?? 'Échec de la création du lot.';
						errorMessage = message;
						toast.error(message);
						return;
					}
					if (result.type === 'error') {
						const message = result.error?.message ?? 'Erreur serveur lors de la création du lot.';
						errorMessage = message;
						toast.error(message);
					}
				};
			}}
			class="flex flex-col gap-4"
		>
			<input type="hidden" name="property_id" value={propertyId} />
			<input type="hidden" name="lot_type" value={lotType} />

			<Field.Group>
				<Field.Field>
					<Field.Label>Bien immobilier parent</Field.Label>
					<Select.Root
						type="single"
						value={propertyId}
						onValueChange={(next) => {
							if (next) propertyId = next;
						}}
					>
						<Select.Trigger class="w-full">
							{properties.find((property) => property.id === propertyId)?.label ??
								'Sélectionner un bien'}
						</Select.Trigger>
						<Select.Content>
							<Select.Group>
								{#each properties as property (property.id)}
									<Select.Item value={property.id} label={property.label}>
										{property.label}
									</Select.Item>
								{/each}
							</Select.Group>
						</Select.Content>
					</Select.Root>
				</Field.Field>

				<Field.Field>
					<Field.Label for="create-lot-label">Référence / Numéro du lot</Field.Label>
					<Input
						id="create-lot-label"
						name="label"
						placeholder="Lot A2 — T2 Cour intérieure"
						bind:value={label}
						required
					/>
				</Field.Field>

				<div class="grid gap-3 sm:grid-cols-2">
					<Field.Field>
						<Field.Label>Type de lot</Field.Label>
						<Select.Root
							type="single"
							value={lotType}
							onValueChange={(next) => {
								if (next) lotType = next;
							}}
						>
							<Select.Trigger class="w-full">
								{formatLotTypeLabel(lotType)}
							</Select.Trigger>
							<Select.Content>
								<Select.Group>
									{#each LOT_TYPE_OPTIONS as option (option.value)}
										<Select.Item value={option.value} label={option.label}>
											{option.label}
										</Select.Item>
									{/each}
								</Select.Group>
							</Select.Content>
						</Select.Root>
					</Field.Field>

					<Field.Field>
						<Field.Label for="create-lot-floor">Étage / Accès</Field.Label>
						<Input
							id="create-lot-floor"
							name="floor"
							placeholder="2ème étage droite"
							bind:value={floor}
						/>
					</Field.Field>
				</div>

				<div class="grid gap-3 sm:grid-cols-2">
					<Field.Field>
						<Field.Label for="create-lot-surface">Surface Loi Carrez / Habitable (m²)</Field.Label>
						<Input
							id="create-lot-surface"
							name="surface_m2"
							type="number"
							step="1"
							min="0"
							bind:value={surfaceM2}
							required
						/>
					</Field.Field>

					<Field.Field>
						<Field.Label for="create-lot-share">Quote-part (Tantièmes / 1000)</Field.Label>
						<Input
							id="create-lot-share"
							name="ownership_share"
							type="number"
							min="1"
							bind:value={ownershipShare}
							required
						/>
					</Field.Field>
				</div>
			</Field.Group>

			<Dialog.Footer>
				<Button
					type="button"
					variant="outline"
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
					Enregistrer le lot
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
