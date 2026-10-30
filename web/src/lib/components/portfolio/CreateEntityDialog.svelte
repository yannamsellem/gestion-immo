<script lang="ts">
	import { enhance } from '$app/forms';
	import { invalidateAll } from '$app/navigation';
	import { resolve } from '$app/paths';
	import IconAlertTriangle from '@tabler/icons-svelte/icons/alert-triangle';
	import IconBuildingSkyscraper from '@tabler/icons-svelte/icons/building-skyscraper';
	import IconLogout from '@tabler/icons-svelte/icons/logout';
	import { toast } from 'svelte-sonner';
	import ThemeToggle from '#lib/components/layout/ThemeToggle.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Dialog from '#lib/components/ui/dialog/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import * as Select from '#lib/components/ui/select/index.js';
	import { Separator } from '#lib/components/ui/separator/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import { workspaceStore } from '#lib/core/workspace/workspace.svelte';
	import { ENTITY_TYPE_OPTIONS, formatEntityTypeLabel } from '#lib/modules/portfolio/labels';
	import type { EntityType, LegalEntityDTO } from '#lib/modules/portfolio/types';

	let {
		open = $bindable(false),
		persistent = false,
		onCreated
	}: {
		open?: boolean;
		persistent?: boolean;
		onCreated?: (entity: LegalEntityDTO) => void;
	} = $props();

	let name = $state('');
	let entityType = $state<EntityType>('SCI_IS');
	let siret = $state('');
	let nafCode = $state('6820B');
	let address = $state('');
	let shareCapitalEuro = $state('1000,00');
	let totalShares = $state('100');
	let isSubmitting = $state(false);
	let errorMessage = $state<string | null>(null);

	function resetForm(): void {
		name = '';
		entityType = 'SCI_IS';
		siret = '';
		nafCode = '6820B';
		address = '';
		shareCapitalEuro = '1000,00';
		totalShares = '100';
		errorMessage = null;
	}
</script>

<Dialog.Root
	open={persistent ? true : open}
	onOpenChange={(next) => {
		if (!persistent) {
			open = next;
			if (!next) {
				errorMessage = null;
			}
		}
	}}
>
	<Dialog.Content
		class="max-h-[90vh] overflow-y-auto sm:max-w-lg"
		showCloseButton={!persistent}
		escapeKeydownBehavior={persistent ? 'ignore' : 'close'}
		interactOutsideBehavior={persistent ? 'ignore' : 'close'}
	>
		<Dialog.Header>
			<div class="flex items-center gap-2.5">
				<div
					class="flex size-9 shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground"
				>
					<IconBuildingSkyscraper class="size-5" />
				</div>
				<div class="flex flex-col gap-0.5">
					<Dialog.Title class="text-base">
						{#if persistent}
							Bienvenue — Créez votre première entité juridique
						{:else}
							Nouvelle entité juridique
						{/if}
					</Dialog.Title>
					<Dialog.Description>
						{#if persistent}
							Toute la gestion patrimoniale, locative et comptable s’articule autour d’une structure
							porteuse (SCI, SARL, SAS, nom propre ou indivision).
						{:else}
							Configurez les informations légales et le capital social initial de la société ou
							détention.
						{/if}
					</Dialog.Description>
				</div>
			</div>
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
			action="{resolve('/(app)/entities')}?/createEntity"
			use:enhance={() => {
				isSubmitting = true;
				errorMessage = null;
				return async ({ result }) => {
					try {
						if (result.type === 'success') {
							const payload = result.data as
								{ createdEntity?: LegalEntityDTO; successMessage?: string } | undefined;
							toast.success(payload?.successMessage ?? 'Entité juridique créée.');
							if (payload?.createdEntity) {
								await workspaceStore.setEntityId(payload.createdEntity.id);
								onCreated?.(payload.createdEntity);
							} else {
								await invalidateAll();
							}
							if (!persistent) {
								resetForm();
								open = false;
							}
							return;
						}

						if (result.type === 'failure') {
							const payload = result.data as { actionError?: string } | undefined;
							const message = payload?.actionError ?? 'Impossible de créer l’entité juridique.';
							errorMessage = message;
							toast.error(message);
							return;
						}

						if (result.type === 'error') {
							const message =
								result.error?.message ?? 'Erreur serveur lors de la création de l’entité.';
							errorMessage = message;
							toast.error(message);
						}
					} finally {
						isSubmitting = false;
					}
				};
			}}
			class="flex flex-col gap-4"
		>
			<input type="hidden" name="entity_type" value={entityType} />

			<Field.Group>
				<Field.Field>
					<Field.Label for="create-entity-name">Dénomination sociale / Nom</Field.Label>
					<Input
						id="create-entity-name"
						name="name"
						placeholder="SCI Patrimoine Haussmann"
						bind:value={name}
						required
					/>
				</Field.Field>

				<Field.Field>
					<Field.Label>Forme juridique & régime fiscal</Field.Label>
					<Select.Root
						type="single"
						value={entityType}
						onValueChange={(next) => {
							if (next) entityType = next;
						}}
					>
						<Select.Trigger class="w-full">
							{formatEntityTypeLabel(entityType)}
						</Select.Trigger>
						<Select.Content>
							<Select.Group>
								{#each ENTITY_TYPE_OPTIONS as option (option.value)}
									<Select.Item value={option.value} label={option.label}>
										{option.label}
									</Select.Item>
								{/each}
							</Select.Group>
						</Select.Content>
					</Select.Root>
				</Field.Field>

				<div class="grid gap-3 sm:grid-cols-2">
					<Field.Field>
						<Field.Label for="create-entity-siret">N° SIRET (14 chiffres)</Field.Label>
						<Input
							id="create-entity-siret"
							name="siret"
							placeholder="89234567800014"
							bind:value={siret}
						/>
					</Field.Field>

					<Field.Field>
						<Field.Label for="create-entity-naf">Code APE / NAF</Field.Label>
						<Input
							id="create-entity-naf"
							name="naf_code"
							placeholder="6820B"
							bind:value={nafCode}
						/>
					</Field.Field>
				</div>

				<Field.Field>
					<Field.Label for="create-entity-address">Adresse du siège social</Field.Label>
					<Input
						id="create-entity-address"
						name="address"
						placeholder="14 boulevard Haussmann, 75009 Paris"
						bind:value={address}
						required
					/>
				</Field.Field>

				<div class="grid gap-3 sm:grid-cols-2">
					<Field.Field>
						<Field.Label for="create-entity-capital">Capital social (€)</Field.Label>
						<Input
							id="create-entity-capital"
							name="share_capital_euro"
							type="text"
							inputmode="decimal"
							placeholder="1000,00"
							bind:value={shareCapitalEuro}
							required
						/>
					</Field.Field>

					<Field.Field>
						<Field.Label for="create-entity-shares">Nombre total de parts</Field.Label>
						<Input
							id="create-entity-shares"
							name="total_shares"
							type="number"
							min="1"
							bind:value={totalShares}
							required
						/>
					</Field.Field>
				</div>
			</Field.Group>

			<Dialog.Footer class={persistent ? 'sm:justify-stretch' : ''}>
				{#if !persistent}
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
				{/if}
				<Button type="submit" class={persistent ? 'w-full' : ''} disabled={isSubmitting}>
					{#if isSubmitting}
						<Spinner data-icon="inline-start" />
					{/if}
					{#if persistent}
						Créer l’entité juridique et démarrer
					{:else}
						Créer l’entité juridique
					{/if}
				</Button>
			</Dialog.Footer>
		</form>

		{#if persistent}
			<Separator />
			<div class="flex items-center justify-between gap-2 text-xs text-muted-foreground">
				<span>Connecté avec un autre compte ?</span>
				<div class="flex items-center gap-1.5">
					<ThemeToggle />
					<form method="POST" action={resolve('/(auth)/logout')} use:enhance>
						<Button type="submit" variant="ghost" size="sm">
							<IconLogout data-icon="inline-start" />
							Se déconnecter
						</Button>
					</form>
				</div>
			</div>
		{/if}
	</Dialog.Content>
</Dialog.Root>
