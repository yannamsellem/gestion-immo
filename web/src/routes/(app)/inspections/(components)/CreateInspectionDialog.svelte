<script lang="ts">
	import { enhance } from '$app/forms';
	import { invalidateAll } from '$app/navigation';
	import { resolve } from '$app/paths';
	import IconAlertTriangle from '@tabler/icons-svelte/icons/alert-triangle';
	import IconInfoCircle from '@tabler/icons-svelte/icons/info-circle';
	import { toast } from 'svelte-sonner';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Dialog from '#lib/components/ui/dialog/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import * as Select from '#lib/components/ui/select/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import { formatIsoDateFr } from '#lib/core/formatters/date';
	import {
		formatInspectionTypeLabel,
		INSPECTION_TYPE_OPTIONS
	} from '#lib/modules/inspection/labels';
	import type {
		InspectionDTO,
		InspectionTemplateDTO,
		InspectionType
	} from '#lib/modules/inspection/types';
	import type { LeaseDTO, TenantDTO } from '#lib/modules/lease/types';

	const DEFAULT_KEYS_DESC = 'Clés porte blindée + badge immeuble';

	let {
		open = $bindable(false),
		leases,
		tenants,
		templates,
		inspections,
		onCreated
	}: {
		open?: boolean;
		leases: LeaseDTO[];
		tenants: TenantDTO[];
		templates: InspectionTemplateDTO[];
		inspections: InspectionDTO[];
		onCreated?: (inspection: InspectionDTO) => void;
	} = $props();

	let leaseId = $state('');
	let templateId = $state('');
	let inspectionType = $state<InspectionType>('MOVE_IN');
	let entryInspectionId = $state('');
	let keysCount = $state('2');
	let keysDesc = $state(DEFAULT_KEYS_DESC);
	let isSubmitting = $state(false);
	let errorMessage = $state<string | null>(null);

	const tenantMap = $derived(new Map(tenants.map((tenant) => [tenant.id, tenant])));
	const leaseMap = $derived(new Map(leases.map((lease) => [lease.id, lease])));
	const moveInInspectionsForLease = $derived(
		inspections.filter((insp) => insp.lease_id === leaseId && insp.type === 'MOVE_IN')
	);

	$effect(() => {
		if (open) {
			if (leases.length > 0 && !leases.some((lease) => lease.id === leaseId)) {
				leaseId = leases[0].id;
			}
			if (templates.length > 0 && !templates.some((tpl) => tpl.id === templateId)) {
				templateId = templates[0].id;
			}
		}
	});

	$effect(() => {
		if (
			entryInspectionId &&
			!moveInInspectionsForLease.some((insp) => insp.id === entryInspectionId)
		) {
			entryInspectionId = '';
		}
	});

	function formatLeaseDisplayLabel(lease: LeaseDTO): string {
		const tenant = tenantMap.get(lease.tenant_id);
		return `${tenant?.full_name ?? 'Locataire'} (Bail ${lease.id.slice(0, 6)})`;
	}

	function resetForm(): void {
		inspectionType = 'MOVE_IN';
		entryInspectionId = '';
		keysCount = '2';
		keysDesc = DEFAULT_KEYS_DESC;
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
	<Dialog.Content class="max-h-[90vh] overflow-y-auto sm:max-w-lg">
		<Dialog.Header>
			<Dialog.Title>Démarrer un état des lieux contradictoire</Dialog.Title>
			<Dialog.Description>
				Sélectionnez un bail actif et un modèle de pièces pour initialiser la grille d’entrée ou de
				sortie.
			</Dialog.Description>
		</Dialog.Header>

		{#if templates.length === 0}
			<Alert.Root>
				<IconInfoCircle />
				<Alert.Title>Aucun modèle d’état des lieux disponible</Alert.Title>
				<Alert.Description>
					Configurez d’abord une grille de pièces dans
					<a
						href={resolve('/(app)/inspections/templates')}
						class="font-medium underline underline-offset-4"
					>
						Modèles d’EDL
					</a>
					avant de démarrer un état des lieux.
				</Alert.Description>
			</Alert.Root>
		{/if}

		{#if leases.length === 0}
			<Alert.Root>
				<IconInfoCircle />
				<Alert.Title>Aucun bail actif disponible</Alert.Title>
				<Alert.Description>
					Créez d’abord un contrat de location dans
					<a href={resolve('/(app)/leases')} class="font-medium underline underline-offset-4">
						Baux actifs
					</a>.
				</Alert.Description>
			</Alert.Root>
		{/if}

		{#if errorMessage}
			<Alert.Root variant="destructive">
				<IconAlertTriangle />
				<Alert.Title>Création impossible</Alert.Title>
				<Alert.Description>{errorMessage}</Alert.Description>
			</Alert.Root>
		{/if}

		<form
			method="POST"
			action="?/createInspection"
			use:enhance={() => {
				isSubmitting = true;
				errorMessage = null;
				return async ({ result }) => {
					isSubmitting = false;
					if (result.type === 'success') {
						const payload = result.data as
							{ createdInspection?: InspectionDTO; successMessage?: string } | undefined;
						toast.success(payload?.successMessage ?? 'État des lieux initialisé.');
						resetForm();
						open = false;
						await invalidateAll();
						if (payload?.createdInspection) {
							onCreated?.(payload.createdInspection);
						}
						return;
					}
					if (result.type === 'failure') {
						const payload = result.data as { actionError?: string } | undefined;
						const message = payload?.actionError ?? 'Impossible de créer l’état des lieux.';
						errorMessage = message;
						toast.error(message);
						return;
					}
					if (result.type === 'error') {
						const message =
							result.error?.message ?? 'Erreur serveur lors de la création de l’état des lieux.';
						errorMessage = message;
						toast.error(message);
					}
				};
			}}
			class="flex flex-col gap-4"
		>
			<input type="hidden" name="lease_id" value={leaseId} />
			<input type="hidden" name="template_id" value={templateId} />
			<input type="hidden" name="type" value={inspectionType} />

			<Field.Group>
				<Field.Field>
					<Field.Label>Bail locatif</Field.Label>
					<Select.Root
						type="single"
						value={leaseId}
						onValueChange={(next) => {
							if (next) leaseId = next;
						}}
					>
						<Select.Trigger class="w-full">
							{#if leaseId && leaseMap.get(leaseId)}
								{formatLeaseDisplayLabel(leaseMap.get(leaseId)!)}
							{:else}
								Sélectionner un bail
							{/if}
						</Select.Trigger>
						<Select.Content>
							<Select.Group>
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
						<Field.Label>Type d’état des lieux</Field.Label>
						<Select.Root
							type="single"
							value={inspectionType}
							onValueChange={(next) => {
								if (next) inspectionType = next;
							}}
						>
							<Select.Trigger class="w-full">
								{formatInspectionTypeLabel(inspectionType)}
							</Select.Trigger>
							<Select.Content>
								<Select.Group>
									{#each INSPECTION_TYPE_OPTIONS as option (option.value)}
										<Select.Item value={option.value} label={option.label}>
											{option.value === 'MOVE_IN'
												? 'Entrant (Remise des clés)'
												: 'Sortant (Comparatif)'}
										</Select.Item>
									{/each}
								</Select.Group>
							</Select.Content>
						</Select.Root>
					</Field.Field>

					<Field.Field>
						<Field.Label>Modèle de grille</Field.Label>
						<Select.Root
							type="single"
							value={templateId}
							onValueChange={(next) => {
								if (next) templateId = next;
							}}
						>
							<Select.Trigger class="w-full">
								{templates.find((tpl) => tpl.id === templateId)?.name ?? 'Choisir un modèle'}
							</Select.Trigger>
							<Select.Content>
								<Select.Group>
									{#each templates as template (template.id)}
										<Select.Item value={template.id} label={template.name}>
											{template.name} ({template.rooms.length} pièces)
										</Select.Item>
									{/each}
								</Select.Group>
							</Select.Content>
						</Select.Root>
					</Field.Field>
				</div>

				{#if inspectionType === 'MOVE_OUT'}
					<input type="hidden" name="entry_inspection_id" value={entryInspectionId} />
					<Field.Field>
						<Field.Label>État des lieux d’entrée de référence (comparatif automatique)</Field.Label>
						<Select.Root
							type="single"
							value={entryInspectionId}
							onValueChange={(next) => {
								if (next !== undefined) entryInspectionId = next;
							}}
						>
							<Select.Trigger class="w-full">
								{#if entryInspectionId}
									{@const entryInsp = moveInInspectionsForLease.find(
										(item) => item.id === entryInspectionId
									)}
									{entryInsp
										? `EDL Entrant du ${formatIsoDateFr(entryInsp.inspection_date)} (${entryInsp.id.slice(0, 8)})`
										: entryInspectionId}
								{:else}
									Aucun EDL d’entrée rattaché (rapport autonome)
								{/if}
							</Select.Trigger>
							<Select.Content>
								<Select.Group>
									<Select.Item value="" label="Aucun (rapport autonome)">
										Aucun EDL d’entrée rattaché (rapport autonome)
									</Select.Item>
									{#each moveInInspectionsForLease as entryInsp (entryInsp.id)}
										<Select.Item
											value={entryInsp.id}
											label={`EDL Entrant du ${formatIsoDateFr(entryInsp.inspection_date)}`}
										>
											EDL Entrant du {formatIsoDateFr(entryInsp.inspection_date)} ({entryInsp.items
												.length} éléments)
										</Select.Item>
									{/each}
								</Select.Group>
							</Select.Content>
						</Select.Root>
					</Field.Field>
				{/if}

				<div class="grid gap-3 sm:grid-cols-2">
					<Field.Field>
						<Field.Label for="create-edl-keys-count">Nombre de clés remises</Field.Label>
						<Input
							id="create-edl-keys-count"
							name="keys_count"
							type="number"
							min="0"
							bind:value={keysCount}
						/>
					</Field.Field>
					<Field.Field>
						<Field.Label for="create-edl-keys-desc">Détail des clés / badges</Field.Label>
						<Input id="create-edl-keys-desc" name="keys_description" bind:value={keysDesc} />
					</Field.Field>
				</div>
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
				<Button
					type="submit"
					disabled={isSubmitting ||
						leases.length === 0 ||
						templates.length === 0 ||
						!leaseId ||
						!templateId}
				>
					{#if isSubmitting}
						<Spinner data-icon="inline-start" />
					{/if}
					Ouvrir la grille d’état des lieux →
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
