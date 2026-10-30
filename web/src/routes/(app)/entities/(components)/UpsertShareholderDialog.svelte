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
	let fullName = $state('');
	let sharesCount = $state('50');
	let currentAccountEuro = $state('0,00');
	let remunerationEuro = $state('0,00');
	let isSubmitting = $state(false);
	let errorMessage = $state<string | null>(null);

	$effect(() => {
		if (open) {
			entityId = defaultEntityId || entities[0]?.id || '';
		}
	});

	function resetForm(): void {
		fullName = '';
		sharesCount = '50';
		currentAccountEuro = '0,00';
		remunerationEuro = '0,00';
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
			<Dialog.Title>Ajouter / modifier un associé</Dialog.Title>
			<Dialog.Description>
				La quote-part en points de base est calculée automatiquement selon le nombre total de parts
				de l’entité.
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
			action="?/upsertShareholder"
			use:enhance={() => {
				isSubmitting = true;
				errorMessage = null;
				return async ({ result }) => {
					isSubmitting = false;
					if (result.type === 'success') {
						const payload = result.data as { successMessage?: string } | undefined;
						toast.success(payload?.successMessage ?? 'Associé enregistré.');
						resetForm();
						open = false;
						await invalidateAll();
						return;
					}
					if (result.type === 'failure') {
						const payload = result.data as { actionError?: string } | undefined;
						const message = payload?.actionError ?? 'Impossible d’enregistrer l’associé.';
						errorMessage = message;
						toast.error(message);
						return;
					}
					if (result.type === 'error') {
						const message =
							result.error?.message ?? 'Erreur serveur lors de l’enregistrement de l’associé.';
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
							{entities.find((entity) => entity.id === entityId)?.name ?? 'Sélectionner'}
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
					<Field.Label for="shareholder-name"
						>Nom complet ou raison sociale de l’associé</Field.Label
					>
					<Input
						id="shareholder-name"
						name="full_name"
						placeholder="Jean Dupont"
						bind:value={fullName}
						required
					/>
				</Field.Field>

				<Field.Field>
					<Field.Label for="shareholder-shares">Nombre de parts détenues</Field.Label>
					<Input
						id="shareholder-shares"
						name="shares_count"
						type="number"
						min="1"
						bind:value={sharesCount}
						required
					/>
				</Field.Field>

				<div class="grid gap-3 sm:grid-cols-2">
					<Field.Field>
						<Field.Label for="shareholder-cc">Solde compte courant 455 (€)</Field.Label>
						<Input
							id="shareholder-cc"
							name="current_account_euro"
							type="text"
							inputmode="decimal"
							bind:value={currentAccountEuro}
						/>
					</Field.Field>

					<Field.Field>
						<Field.Label for="shareholder-rem">Rémunération versée (€)</Field.Label>
						<Input
							id="shareholder-rem"
							name="remuneration_euro"
							type="text"
							inputmode="decimal"
							bind:value={remunerationEuro}
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
					Enregistrer l’associé
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
