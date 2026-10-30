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

	const DEFAULT_DEPOSIT_LABEL = 'Remise espèces en banque (530 → 512)';

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
	let amountEuro = $state('');
	let label = $state(DEFAULT_DEPOSIT_LABEL);
	let entryDate = $state(new Date().toISOString().slice(0, 10));
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
		amountEuro = '';
		label = DEFAULT_DEPOSIT_LABEL;
		entryDate = new Date().toISOString().slice(0, 10);
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
			<Dialog.Title>Remise d’espèces en banque (PCG 530 → 512)</Dialog.Title>
			<Dialog.Description>
				Crédite le compte Caisse (530) et débite le compte Banque (512) lors du dépôt des espèces.
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
			action="?/cashDeposit"
			use:enhance={() => {
				isSubmitting = true;
				errorMessage = null;
				return async ({ result }) => {
					isSubmitting = false;
					if (result.type === 'success') {
						const payload = result.data as { successMessage?: string } | undefined;
						toast.success(
							payload?.successMessage ??
								'Remise d’espèces en banque enregistrée (écriture 530 → 512).'
						);
						resetForm();
						open = false;
						await invalidateAll();
						return;
					}
					if (result.type === 'failure') {
						const payload = result.data as { actionError?: string } | undefined;
						const message = payload?.actionError ?? 'Impossible d’enregistrer la remise d’espèces.';
						errorMessage = message;
						toast.error(message);
						return;
					}
					if (result.type === 'error') {
						const message =
							result.error?.message ?? 'Erreur serveur lors de l’enregistrement de la remise.';
						errorMessage = message;
						toast.error(message);
					}
				};
			}}
			class="flex flex-col gap-4"
		>
			<input type="hidden" name="legal_entity_id" value={entityId} />

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

				<div class="grid gap-3 sm:grid-cols-2">
					<Field.Field>
						<Field.Label for="cash-deposit-amount">Montant déposé (€)</Field.Label>
						<Input
							id="cash-deposit-amount"
							name="amount_euro"
							type="text"
							inputmode="decimal"
							placeholder="500,00"
							bind:value={amountEuro}
							required
						/>
					</Field.Field>

					<Field.Field>
						<Field.Label for="cash-deposit-date">Date du dépôt</Field.Label>
						<Input
							id="cash-deposit-date"
							name="entry_date"
							type="date"
							bind:value={entryDate}
							required
						/>
					</Field.Field>
				</div>

				<Field.Field>
					<Field.Label for="cash-deposit-label">Libellé de l’écriture</Field.Label>
					<Input id="cash-deposit-label" name="label" type="text" bind:value={label} required />
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
					Enregistrer la remise
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
