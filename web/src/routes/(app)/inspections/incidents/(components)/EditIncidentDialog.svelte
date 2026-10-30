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
	import { Separator } from '#lib/components/ui/separator/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import { formatIsoDateFr } from '#lib/core/formatters/date';
	import {
		formatIncidentStatusLabel,
		INCIDENT_STATUS_OPTIONS
	} from '#lib/modules/inspection/labels';
	import type { IncidentDTO, IncidentStatus } from '#lib/modules/inspection/types';

	let {
		open = $bindable(false),
		incident,
		onUpdated
	}: {
		open?: boolean;
		incident: IncidentDTO | null;
		onUpdated?: (incident: IncidentDTO) => void;
	} = $props();

	let status = $state<IncidentStatus>('OPEN');
	let claimNumber = $state('');
	let costEuro = $state('0,00');
	let newObservation = $state('');
	let isSubmitting = $state(false);
	let errorMessage = $state<string | null>(null);
	let syncedIncidentId = $state<string | null>(null);

	$effect(() => {
		if (open && incident && syncedIncidentId !== incident.id) {
			status = incident.status;
			claimNumber = incident.insurance_claim_number ?? '';
			costEuro = (incident.estimated_cost_cents / 100).toFixed(2).replace('.', ',');
			newObservation = '';
			errorMessage = null;
			syncedIncidentId = incident.id;
		}
		if (!open) {
			syncedIncidentId = null;
		}
	});
</script>

<Dialog.Root
	bind:open
	onOpenChange={(next) => {
		if (!next) {
			errorMessage = null;
		}
	}}
>
	<Dialog.Content class="max-h-[90vh] overflow-y-auto sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>{incident?.title ?? 'Suivi d’incident'}</Dialog.Title>
			<Dialog.Description>
				Déclaré le {formatIsoDateFr(incident?.occurred_at)}
			</Dialog.Description>
		</Dialog.Header>

		{#if errorMessage}
			<Alert.Root variant="destructive">
				<IconAlertTriangle />
				<Alert.Title>Mise à jour impossible</Alert.Title>
				<Alert.Description>{errorMessage}</Alert.Description>
			</Alert.Root>
		{/if}

		{#if incident}
			<form
				method="POST"
				action="?/updateIncident"
				use:enhance={() => {
					isSubmitting = true;
					errorMessage = null;
					return async ({ result }) => {
						isSubmitting = false;
						if (result.type === 'success') {
							const payload = result.data as
								{ updatedIncident?: IncidentDTO; successMessage?: string } | undefined;
							toast.success(payload?.successMessage ?? 'Incident mis à jour.');
							newObservation = '';
							open = false;
							await invalidateAll();
							if (payload?.updatedIncident) {
								onUpdated?.(payload.updatedIncident);
							}
							return;
						}
						if (result.type === 'failure') {
							const payload = result.data as { actionError?: string } | undefined;
							const message = payload?.actionError ?? 'Impossible de mettre à jour l’incident.';
							errorMessage = message;
							toast.error(message);
							return;
						}
						if (result.type === 'error') {
							const message =
								result.error?.message ?? 'Erreur serveur lors de la mise à jour de l’incident.';
							errorMessage = message;
							toast.error(message);
						}
					};
				}}
				class="flex flex-col gap-4"
			>
				<input type="hidden" name="incident_id" value={incident.id} />
				<input type="hidden" name="status" value={status} />

				<Field.Group>
					<Field.Field>
						<Field.Label>Statut d’avancement</Field.Label>
						<Select.Root
							type="single"
							value={status}
							onValueChange={(next) => {
								if (next) status = next;
							}}
						>
							<Select.Trigger class="w-full">
								{formatIncidentStatusLabel(status)}
							</Select.Trigger>
							<Select.Content>
								<Select.Group>
									{#each INCIDENT_STATUS_OPTIONS as option (option.value)}
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
							<Field.Label for="edit-incident-claim">N° sinistre assurance</Field.Label>
							<Input
								id="edit-incident-claim"
								name="insurance_claim_number"
								bind:value={claimNumber}
							/>
						</Field.Field>

						<Field.Field>
							<Field.Label for="edit-incident-cost">Coût estimé (€)</Field.Label>
							<Input
								id="edit-incident-cost"
								name="estimated_cost_euro"
								type="text"
								inputmode="decimal"
								bind:value={costEuro}
							/>
						</Field.Field>
					</div>

					<Field.Field>
						<Field.Label for="edit-incident-obs">Ajouter une note de suivi</Field.Label>
						<Input
							id="edit-incident-obs"
							name="add_observation"
							placeholder="Devis validé / intervention planifiée..."
							bind:value={newObservation}
						/>
					</Field.Field>
				</Field.Group>

				{#if incident.observations && incident.observations.length > 0}
					<Separator />
					<div class="flex flex-col gap-2">
						<span class="text-xs font-semibold uppercase">Historique des observations</span>
						{#each incident.observations as obs, index (`${incident.id}-obs-${index}`)}
							<div class="rounded-md border bg-muted/40 p-2.5 text-xs">
								{obs}
							</div>
						{/each}
					</div>
				{/if}

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
						Enregistrer les modifications
					</Button>
				</Dialog.Footer>
			</form>
		{/if}
	</Dialog.Content>
</Dialog.Root>
