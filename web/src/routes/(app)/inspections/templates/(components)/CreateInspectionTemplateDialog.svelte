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
	import type { InspectionTemplateDTO } from '#lib/modules/inspection/types';

	const DEFAULT_ROOMS_TEXT = 'Entrée, Séjour, Cuisine, Chambre, Salle de bain, WC';
	const DEFAULT_ELEMENTS_TEXT = 'Sol, Murs, Plafond, Fenêtres, Électricité, Plomberie';

	let {
		open = $bindable(false),
		onCreated
	}: {
		open?: boolean;
		onCreated?: (template: InspectionTemplateDTO) => void;
	} = $props();

	let name = $state('');
	let roomsText = $state(DEFAULT_ROOMS_TEXT);
	let elementsText = $state(DEFAULT_ELEMENTS_TEXT);
	let isSubmitting = $state(false);
	let errorMessage = $state<string | null>(null);

	function resetForm(): void {
		name = '';
		roomsText = DEFAULT_ROOMS_TEXT;
		elementsText = DEFAULT_ELEMENTS_TEXT;
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
			<Dialog.Title>Créer un modèle d’état des lieux</Dialog.Title>
			<Dialog.Description>
				Définissez la liste des pièces et des éléments inspectés par défaut.
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
			action="?/createTemplate"
			use:enhance={() => {
				isSubmitting = true;
				errorMessage = null;
				return async ({ result }) => {
					isSubmitting = false;
					if (result.type === 'success') {
						const payload = result.data as
							{ createdTemplate?: InspectionTemplateDTO; successMessage?: string } | undefined;
						toast.success(payload?.successMessage ?? 'Modèle d’état des lieux créé.');
						resetForm();
						open = false;
						await invalidateAll();
						if (payload?.createdTemplate) {
							onCreated?.(payload.createdTemplate);
						}
						return;
					}
					if (result.type === 'failure') {
						const payload = result.data as { actionError?: string } | undefined;
						const message = payload?.actionError ?? 'Erreur lors de la création du modèle.';
						errorMessage = message;
						toast.error(message);
						return;
					}
					if (result.type === 'error') {
						const message =
							result.error?.message ?? 'Erreur serveur lors de la création du modèle.';
						errorMessage = message;
						toast.error(message);
					}
				};
			}}
			class="flex flex-col gap-4"
		>
			<Field.Group>
				<Field.Field>
					<Field.Label for="create-edl-tpl-name">Nom du modèle</Field.Label>
					<Input
						id="create-edl-tpl-name"
						name="name"
						placeholder="Appartement T2 Standard"
						bind:value={name}
						required
					/>
				</Field.Field>

				<Field.Field>
					<Field.Label for="create-edl-tpl-rooms">Pièces (séparées par des virgules)</Field.Label>
					<Input id="create-edl-tpl-rooms" name="rooms_text" bind:value={roomsText} required />
				</Field.Field>

				<Field.Field>
					<Field.Label for="create-edl-tpl-elements">
						Éléments par pièce (séparés par des virgules)
					</Field.Label>
					<Input
						id="create-edl-tpl-elements"
						name="elements_text"
						bind:value={elementsText}
						required
					/>
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
				<Button type="submit" disabled={isSubmitting}>
					{#if isSubmitting}
						<Spinner data-icon="inline-start" />
					{/if}
					Créer le modèle
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
