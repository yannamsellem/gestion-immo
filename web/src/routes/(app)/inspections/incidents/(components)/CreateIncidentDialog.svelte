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
	import type { IncidentDTO } from '#lib/modules/inspection/types';
	import type { LeaseDTO, TenantDTO } from '#lib/modules/lease/types';
	import type { PropertyDTO } from '#lib/modules/portfolio/types';

	let {
		open = $bindable(false),
		properties,
		leases,
		tenants,
		onCreated
	}: {
		open?: boolean;
		properties: PropertyDTO[];
		leases: LeaseDTO[];
		tenants: TenantDTO[];
		onCreated?: (incident: IncidentDTO) => void;
	} = $props();

	let title = $state('');
	let propertyId = $state('');
	let leaseId = $state('');
	let claimNumber = $state('');
	let estimatedEuro = $state('0,00');
	let observation = $state('');
	let isSubmitting = $state(false);
	let errorMessage = $state<string | null>(null);

	const propertyMap = $derived(new Map(properties.map((property) => [property.id, property])));
	const tenantMap = $derived(new Map(tenants.map((tenant) => [tenant.id, tenant])));
	const leaseMap = $derived(new Map(leases.map((lease) => [lease.id, lease])));

	$effect(() => {
		if (open && properties.length > 0 && !properties.some((prop) => prop.id === propertyId)) {
			propertyId = properties[0].id;
		}
	});

	function formatLeaseDisplayLabel(lease: LeaseDTO): string {
		const tenant = tenantMap.get(lease.tenant_id);
		return `${tenant?.full_name ?? 'Locataire'} (Bail ${lease.id.slice(0, 6)})`;
	}

	function resetForm(): void {
		title = '';
		leaseId = '';
		claimNumber = '';
		estimatedEuro = '0,00';
		observation = '';
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
			<Dialog.Title>Déclarer un incident ou sinistre</Dialog.Title>
			<Dialog.Description>
				Associez l’incident à un bien ou un bail pour qu’il remonte sur le Cockpit Opérationnel.
			</Dialog.Description>
		</Dialog.Header>

		{#if errorMessage}
			<Alert.Root variant="destructive">
				<IconAlertTriangle />
				<Alert.Title>Déclaration impossible</Alert.Title>
				<Alert.Description>{errorMessage}</Alert.Description>
			</Alert.Root>
		{/if}

		<form
			method="POST"
			action="?/createIncident"
			use:enhance={() => {
				isSubmitting = true;
				errorMessage = null;
				return async ({ result }) => {
					isSubmitting = false;
					if (result.type === 'success') {
						const payload = result.data as
							{ createdIncident?: IncidentDTO; successMessage?: string } | undefined;
						toast.success(payload?.successMessage ?? 'Incident enregistré.');
						resetForm();
						open = false;
						await invalidateAll();
						if (payload?.createdIncident) {
							onCreated?.(payload.createdIncident);
						}
						return;
					}
					if (result.type === 'failure') {
						const payload = result.data as { actionError?: string } | undefined;
						const message = payload?.actionError ?? 'Erreur lors de la création de l’incident.';
						errorMessage = message;
						toast.error(message);
						return;
					}
					if (result.type === 'error') {
						const message =
							result.error?.message ?? 'Erreur serveur lors de la déclaration de l’incident.';
						errorMessage = message;
						toast.error(message);
					}
				};
			}}
			class="flex flex-col gap-4"
		>
			<input type="hidden" name="property_id" value={propertyId} />
			<input type="hidden" name="lease_id" value={leaseId} />

			<Field.Group>
				<Field.Field>
					<Field.Label for="create-incident-title">Intitulé de l’incident</Field.Label>
					<Input
						id="create-incident-title"
						name="title"
						placeholder="Dégât des eaux salle de bain"
						bind:value={title}
						required
					/>
				</Field.Field>

				<Field.Field>
					<Field.Label>Bien immobilier concerné</Field.Label>
					<Select.Root
						type="single"
						value={propertyId}
						onValueChange={(next) => {
							if (next !== undefined) propertyId = next;
						}}
					>
						<Select.Trigger class="w-full">
							{propertyMap.get(propertyId)?.label ?? 'Sélectionner un bien'}
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
					<Field.Label>Bail locatif (optionnel)</Field.Label>
					<Select.Root
						type="single"
						value={leaseId}
						onValueChange={(next) => {
							if (next !== undefined) leaseId = next;
						}}
					>
						<Select.Trigger class="w-full">
							{#if leaseId && leaseMap.get(leaseId)}
								{formatLeaseDisplayLabel(leaseMap.get(leaseId)!)}
							{:else}
								Aucun bail spécifique
							{/if}
						</Select.Trigger>
						<Select.Content>
							<Select.Group>
								<Select.Item value="" label="Aucun bail spécifique">
									Aucun bail spécifique
								</Select.Item>
								{#each leases as lease (lease.id)}
									<Select.Item value={lease.id} label={formatLeaseDisplayLabel(lease)}>
										{formatLeaseDisplayLabel(lease)}
									</Select.Item>
								{/each}
							</Select.Group>
						</Select.Content>
					</Select.Root>
				</Field.Field>

				<div class="grid gap-3 sm:grid-cols-2">
					<Field.Field>
						<Field.Label for="create-incident-claim">N° dossier assurance</Field.Label>
						<Input
							id="create-incident-claim"
							name="insurance_claim_number"
							placeholder="SIN-2026-884"
							bind:value={claimNumber}
						/>
					</Field.Field>

					<Field.Field>
						<Field.Label for="create-incident-cost">Coût estimé (€)</Field.Label>
						<Input
							id="create-incident-cost"
							name="estimated_cost_euro"
							type="text"
							inputmode="decimal"
							placeholder="350,00"
							bind:value={estimatedEuro}
						/>
					</Field.Field>
				</div>

				<Field.Field>
					<Field.Label for="create-incident-obs">Observation initiale</Field.Label>
					<Input
						id="create-incident-obs"
						name="observation"
						placeholder="Plombier mandaté pour recherche de fuite"
						bind:value={observation}
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
					Enregistrer l’incident
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
