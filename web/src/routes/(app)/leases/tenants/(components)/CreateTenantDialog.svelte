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
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import type { TenantDTO } from '#lib/modules/lease/types';

	let {
		open = $bindable(false),
		onCreated
	}: {
		open?: boolean;
		onCreated?: (tenant: TenantDTO) => void;
	} = $props();

	let fullName = $state('');
	let email = $state('');
	let phone = $state('');
	let address = $state('');
	let isSubmitting = $state(false);
	let errorMessage = $state<string | null>(null);

	function resetForm(): void {
		fullName = '';
		email = '';
		phone = '';
		address = '';
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
			<Dialog.Title>Nouveau locataire</Dialog.Title>
			<Dialog.Description>
				Créez une fiche locataire pour l’associer à un bail d’habitation ou commercial.
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
			action="?/createTenant"
			use:enhance={() => {
				isSubmitting = true;
				errorMessage = null;
				return async ({ result }) => {
					isSubmitting = false;
					if (result.type === 'success') {
						const payload = result.data as
							{ createdTenant?: TenantDTO; successMessage?: string } | undefined;
						toast.success(payload?.successMessage ?? 'Locataire enregistré.');
						resetForm();
						open = false;
						await invalidateAll();
						if (payload?.createdTenant) {
							onCreated?.(payload.createdTenant);
						}
						return;
					}
					if (result.type === 'failure') {
						const payload = result.data as { actionError?: string } | undefined;
						const message = payload?.actionError ?? 'Échec de la création du locataire.';
						errorMessage = message;
						toast.error(message);
						return;
					}
					if (result.type === 'error') {
						const message =
							result.error?.message ?? 'Erreur serveur lors de la création du locataire.';
						errorMessage = message;
						toast.error(message);
					}
				};
			}}
			class="flex flex-col gap-4"
		>
			<Field.Group>
				<Field.Field>
					<Field.Label for="create-tenant-name">Nom complet / Raison sociale</Field.Label>
					<Input
						id="create-tenant-name"
						name="full_name"
						placeholder="Claire Martin"
						bind:value={fullName}
						required
					/>
				</Field.Field>

				<Field.Field>
					<Field.Label for="create-tenant-email">Adresse e-mail</Field.Label>
					<Input
						id="create-tenant-email"
						name="email"
						type="email"
						placeholder="claire.martin@exemple.fr"
						bind:value={email}
						required
					/>
				</Field.Field>

				<Field.Field>
					<Field.Label for="create-tenant-phone">Téléphone</Field.Label>
					<Input
						id="create-tenant-phone"
						name="phone"
						type="tel"
						placeholder="06 12 34 56 78"
						bind:value={phone}
					/>
				</Field.Field>

				<Field.Field>
					<Field.Label for="create-tenant-address">Adresse postale</Field.Label>
					<Input
						id="create-tenant-address"
						name="address"
						placeholder="12 rue de Rivoli, 75004 Paris"
						bind:value={address}
					/>
				</Field.Field>
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
					Enregistrer le locataire
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
