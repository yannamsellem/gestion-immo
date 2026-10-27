<script lang="ts">
	import { enhance } from '$app/forms';
	import { page } from '$app/state';
	import IconAlertTriangle from '@tabler/icons-svelte/icons/alert-triangle';
	import IconClipboardCheck from '@tabler/icons-svelte/icons/clipboard-check';
	import IconDownload from '@tabler/icons-svelte/icons/download';
	import IconFileText from '@tabler/icons-svelte/icons/file-text';
	import IconPlus from '@tabler/icons-svelte/icons/plus';
	import IconTool from '@tabler/icons-svelte/icons/tool';
	import { toast } from 'svelte-sonner';
	import PageHeader from '#lib/components/layout/PageHeader.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Dialog from '#lib/components/ui/dialog/index.js';
	import * as Empty from '#lib/components/ui/empty/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import * as Select from '#lib/components/ui/select/index.js';
	import { Separator } from '#lib/components/ui/separator/index.js';
	import * as Sheet from '#lib/components/ui/sheet/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import * as Table from '#lib/components/ui/table/index.js';
	import * as Tabs from '#lib/components/ui/tabs/index.js';
	import { formatIsoDateFr } from '#lib/core/formatters/date';
	import { formatCentsToEuro } from '#lib/core/formatters/money';
	import type {
		ConditionState,
		IncidentDTO,
		IncidentStatus,
		InspectionDTO,
		InspectionItemInput,
		InspectionReportDTO,
		InspectionType
	} from '#lib/modules/inspection/types';
	import type { LeaseDTO } from '#lib/modules/lease/types';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	const INCIDENT_STATUS_OPTIONS: Array<{ value: IncidentStatus; label: string }> = [
		{ value: 'OPEN', label: 'Ouvert' },
		{ value: 'IN_PROGRESS', label: 'En cours' },
		{ value: 'RESOLVED', label: 'Résolu' },
		{ value: 'CLOSED', label: 'Clôturé' }
	];

	const CONDITION_OPTIONS: Array<{ value: ConditionState; label: string }> = [
		{ value: 'NEW', label: 'Neuf' },
		{ value: 'GOOD', label: 'Bon état' },
		{ value: 'FAIR_WEAR', label: 'État d’usage' },
		{ value: 'POOR', label: 'Mauvais état' },
		{ value: 'OUT_OF_SERVICE', label: 'Hors service' }
	];

	let activeTab = $state<'incidents' | 'inspections'>('incidents');

	// Create Incident Sheet
	let createIncidentOpen = $state(false);
	let newIncidentTitle = $state('');
	let newIncidentPropertyId = $state('');
	let newIncidentLeaseId = $state('');
	let newIncidentClaimNumber = $state('');
	let newIncidentEstimatedEuro = $state('0,00');
	let newIncidentObservation = $state('');
	let isCreatingIncident = $state(false);

	// Edit Incident Sheet
	let editIncidentOpen = $state(false);
	let selectedIncident = $state<IncidentDTO | null>(null);
	let editIncidentStatus = $state<IncidentStatus>('OPEN');
	let editIncidentClaimNumber = $state('');
	let editIncidentCostEuro = $state('0,00');
	let editIncidentNewObservation = $state('');
	let isUpdatingIncident = $state(false);
	let autoOpenedIncidentId = $state<string | null>(null);

	// Template Dialog
	let createTemplateOpen = $state(false);
	let newTemplateName = $state('');
	let newTemplateRoomsText = $state('Entrée, Séjour, Cuisine, Chambre, Salle de bain, WC');
	let newTemplateElementsText = $state('Sol, Murs, Plafond, Fenêtres, Électricité, Plomberie');
	let isCreatingTemplate = $state(false);

	// Inspection creation & editing Sheet
	let newInspectionLeaseId = $state('');
	let newInspectionTemplateId = $state('');
	let newInspectionType = $state<InspectionType>('MOVE_IN');
	let newInspectionEntryId = $state('');
	let newInspectionKeysCount = $state('2');
	let newInspectionKeysDesc = $state('Clés porte blindée + badge immeuble');
	let isCreatingInspection = $state(false);

	let activeInspection = $state<InspectionDTO | null>(null);
	let activeInspectionItems = $state<InspectionItemInput[]>([]);
	let activeInspectionReport = $state<InspectionReportDTO | null>(null);
	let inspectionSheetOpen = $state(false);
	let isSavingInspectionItems = $state(false);
	let isGeneratingPdf = $state(false);

	const moveInInspectionsForSelectedLease = $derived(
		data.inspections.filter(
			(insp) => insp.lease_id === newInspectionLeaseId && insp.type === 'MOVE_IN'
		)
	);

	function formatInspectionTypeLabel(type: InspectionType): string {
		if (type === 'MOVE_IN') return 'Entrant';
		if (type === 'MOVE_OUT') return 'Sortant';
		return type;
	}

	$effect(() => {
		const urlTab = page.url.searchParams.get('tab');
		if (urlTab === 'inspections' || urlTab === 'incidents') {
			activeTab = urlTab;
		}
	});

	$effect(() => {
		if (data.properties.length > 0 && !newIncidentPropertyId) {
			newIncidentPropertyId = data.properties[0].id;
		}
		if (data.leases.length > 0 && !newInspectionLeaseId) {
			newInspectionLeaseId = data.leases[0].id;
		}
		if (data.templates.length > 0 && !newInspectionTemplateId) {
			newInspectionTemplateId = data.templates[0].id;
		}
	});

	$effect(() => {
		if (form?.successMessage) {
			toast.success(form.successMessage);
		}
		if (form?.actionError) {
			toast.error(form.actionError);
		}
		if (form?.incidentCreated) {
			createIncidentOpen = false;
			newIncidentTitle = '';
			newIncidentClaimNumber = '';
			newIncidentObservation = '';
		}
		if (form?.updatedIncident) {
			selectedIncident = form.updatedIncident;
			editIncidentNewObservation = '';
		}
		if (form?.createdTemplate) {
			newInspectionTemplateId = form.createdTemplate.id;
			createTemplateOpen = false;
			newTemplateName = '';
		}
		if (form?.createdInspection) {
			openInspectionSheet(form.createdInspection);
		}
		if (form?.updatedInspection) {
			activeInspection = form.updatedInspection;
		}
		if (form?.generatedReport) {
			activeInspectionReport = form.generatedReport;
		}
	});

	// Auto-open incident Sheet when arriving from Cockpit with ?incidentId=...
	$effect(() => {
		if (data.incidents.length === 0) return;
		const urlIncidentId = page.url.searchParams.get('incidentId');
		if (!urlIncidentId || autoOpenedIncidentId === urlIncidentId) return;

		const match = data.incidents.find((inc) => inc.id === urlIncidentId);
		if (match) {
			openEditIncidentSheet(match);
			autoOpenedIncidentId = urlIncidentId;
		}
	});

	const propertyMap = $derived(new Map(data.properties.map((p) => [p.id, p])));
	const tenantMap = $derived(new Map(data.tenants.map((t) => [t.id, t])));
	const leaseMap = $derived(new Map(data.leases.map((l) => [l.id, l])));

	function formatLeaseDisplayLabel(lease: LeaseDTO): string {
		const tenant = tenantMap.get(lease.tenant_id);
		return `${tenant?.full_name ?? 'Locataire'} (Bail ${lease.id.slice(0, 6)})`;
	}

	function formatIncidentStatusLabel(status: IncidentStatus): string {
		return INCIDENT_STATUS_OPTIONS.find((s) => s.value === status)?.label ?? status;
	}

	function formatIncidentStatusVariant(
		status: IncidentStatus
	): 'destructive' | 'default' | 'secondary' | 'outline' {
		if (status === 'OPEN') return 'destructive';
		if (status === 'IN_PROGRESS') return 'default';
		if (status === 'RESOLVED') return 'secondary';
		return 'outline';
	}

	function openEditIncidentSheet(incident: IncidentDTO) {
		selectedIncident = incident;
		editIncidentStatus = incident.status;
		editIncidentClaimNumber = incident.insurance_claim_number ?? '';
		editIncidentCostEuro = (incident.estimated_cost_cents / 100).toFixed(2).replace('.', ',');
		editIncidentNewObservation = '';
		editIncidentOpen = true;
	}

	function openInspectionSheet(inspection: InspectionDTO) {
		activeInspection = inspection;
		activeInspectionReport = null;
		if (inspection.items && inspection.items.length > 0) {
			activeInspectionItems = inspection.items.map((item) => ({
				room_name: item.room_name,
				element_name: item.element_name,
				condition: item.condition,
				observation: item.observation ?? ''
			}));
		} else {
			const tpl = data.templates.find((t) => t.id === inspection.template_id);
			const rooms = tpl?.rooms ?? ['Séjour', 'Cuisine', 'Chambre', 'Salle de bain'];
			const elements = tpl?.default_elements?.length
				? tpl.default_elements.slice(0, 3)
				: ['Sol', 'Murs', 'Équipements'];
			const generated: InspectionItemInput[] = [];
			for (const room of rooms) {
				for (const element of elements) {
					generated.push({
						room_name: room,
						element_name: element,
						condition: 'GOOD',
						observation: ''
					});
				}
			}
			activeInspectionItems = generated;
		}
		inspectionSheetOpen = true;
	}
</script>

<PageHeader
	title="Incidents & États des lieux"
	subtitle="Suivi des sinistres, travaux et états des lieux contradictoires (Entrant / Sortant avec comparatif PDF)"
>
	{#snippet actions()}
		<Button size="sm" variant="outline" onclick={() => (createTemplateOpen = true)}>
			<IconClipboardCheck data-icon="inline-start" />
			Nouveau modèle EDL
		</Button>
		<Button size="sm" onclick={() => (createIncidentOpen = true)}>
			<IconPlus data-icon="inline-start" />
			Déclarer un incident
		</Button>
	{/snippet}
</PageHeader>

{#if data.errorMessage}
	<Alert.Root variant="destructive">
		<IconAlertTriangle class="size-4" />
		<Alert.Title>Erreur</Alert.Title>
		<Alert.Description>{data.errorMessage}</Alert.Description>
	</Alert.Root>
{/if}

<Tabs.Root bind:value={activeTab}>
	<Tabs.List>
		<Tabs.Trigger value="incidents">
			Incidents & Sinistres ({data.incidents.length})
		</Tabs.Trigger>
		<Tabs.Trigger value="inspections">
			États des lieux & Modèles ({data.templates.length})
		</Tabs.Trigger>
	</Tabs.List>

	<!-- Tab 1: Incidents -->
	<Tabs.Content value="incidents" class="mt-4">
		{#if data.incidents.length === 0}
			<Card.Root>
				<Card.Content class="py-10">
					<Empty.Root>
						<Empty.Header>
							<Empty.Media variant="icon">
								<IconTool class="size-5" />
							</Empty.Media>
							<Empty.Title>Aucun incident ou sinistre déclaré</Empty.Title>
							<Empty.Description>
								Enregistrez les dégâts des eaux, pannes ou travaux pour assurer le suivi assurance
								et le chiffrage des réparations.
							</Empty.Description>
						</Empty.Header>
						<Empty.Content>
							<Button onclick={() => (createIncidentOpen = true)}>
								<IconPlus data-icon="inline-start" />
								Déclarer un incident
							</Button>
						</Empty.Content>
					</Empty.Root>
				</Card.Content>
			</Card.Root>
		{:else}
			<Card.Root>
				<Card.Content class="p-0">
					<Table.Root>
						<Table.Header>
							<Table.Row>
								<Table.Head class="pl-4">Date</Table.Head>
								<Table.Head>Intitulé de l'incident</Table.Head>
								<Table.Head>Bien / Bail rattaché</Table.Head>
								<Table.Head>Réf. Sinistre</Table.Head>
								<Table.Head class="text-right">Coût estimé</Table.Head>
								<Table.Head>Statut</Table.Head>
								<Table.Head class="pr-4 text-right">Action</Table.Head>
							</Table.Row>
						</Table.Header>
						<Table.Body>
							{#each data.incidents as incident (incident.id)}
								<Table.Row class="cursor-pointer" onclick={() => openEditIncidentSheet(incident)}>
									<Table.Cell class="pl-4 text-xs">
										{formatIsoDateFr(incident.occurred_at)}
									</Table.Cell>
									<Table.Cell class="font-medium">{incident.title}</Table.Cell>
									<Table.Cell class="text-xs text-muted-foreground">
										{#if incident.property_id && propertyMap.get(incident.property_id)}
											{propertyMap.get(incident.property_id)?.label}
										{:else if incident.lease_id && leaseMap.get(incident.lease_id)}
											{formatLeaseDisplayLabel(leaseMap.get(incident.lease_id)!)}
										{:else}
											—
										{/if}
									</Table.Cell>
									<Table.Cell class="font-mono text-xs">
										{incident.insurance_claim_number || '—'}
									</Table.Cell>
									<Table.Cell class="text-right font-mono tabular-nums">
										{formatCentsToEuro(incident.estimated_cost_cents)}
									</Table.Cell>
									<Table.Cell>
										<Badge variant={formatIncidentStatusVariant(incident.status)}>
											{formatIncidentStatusLabel(incident.status)}
										</Badge>
									</Table.Cell>
									<Table.Cell class="pr-4 text-right">
										<Button
											size="sm"
											variant="outline"
											onclick={(e: MouseEvent) => {
												e.stopPropagation();
												openEditIncidentSheet(incident);
											}}
										>
											Traiter →
										</Button>
									</Table.Cell>
								</Table.Row>
							{/each}
						</Table.Body>
					</Table.Root>
				</Card.Content>
			</Card.Root>
		{/if}
	</Tabs.Content>

	<!-- Tab 2: Move-In / Move-Out Inspections -->
	<Tabs.Content value="inspections" class="mt-4 flex flex-col gap-6">
		<div class="grid gap-6 lg:grid-cols-2">
			<!-- Start new Inspection Card -->
			<Card.Root>
				<Card.Header>
					<Card.Title class="text-sm font-semibold">
						Démarrer un état des lieux contradictoire
					</Card.Title>
					<Card.Description>
						Sélectionnez un bail actif et un modèle de pièces pour saisir l'état d'entrée ou de
						sortie.
					</Card.Description>
				</Card.Header>
				<Card.Content>
					<form
						method="POST"
						action="?/createInspection"
						use:enhance={() => {
							isCreatingInspection = true;
							return async ({ update }) => {
								await update();
								isCreatingInspection = false;
							};
						}}
						class="flex flex-col gap-4"
					>
						<input type="hidden" name="lease_id" value={newInspectionLeaseId} />
						<input type="hidden" name="template_id" value={newInspectionTemplateId} />
						<input type="hidden" name="type" value={newInspectionType} />

						<Field.FieldGroup>
							<Field.Field>
								<Field.Label>Bail locatif</Field.Label>
								<Select.Root
									type="single"
									value={newInspectionLeaseId}
									onValueChange={(v) => {
										if (v) newInspectionLeaseId = v;
									}}
								>
									<Select.Trigger class="w-full">
										{#if newInspectionLeaseId && leaseMap.get(newInspectionLeaseId)}
											{formatLeaseDisplayLabel(leaseMap.get(newInspectionLeaseId)!)}
										{:else}
											Sélectionner un bail
										{/if}
									</Select.Trigger>
									<Select.Content>
										<Select.Group>
											{#each data.leases as lease (lease.id)}
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
									<Field.Label>Type d'état des lieux</Field.Label>
									<Select.Root
										type="single"
										value={newInspectionType}
										onValueChange={(v) => {
											if (v) newInspectionType = v;
										}}
									>
										<Select.Trigger class="w-full">
											{newInspectionType === 'MOVE_IN'
												? 'Entrant (Remise des clés)'
												: 'Sortant (Comparatif)'}
										</Select.Trigger>
										<Select.Content>
											<Select.Group>
												<Select.Item value="MOVE_IN" label="Entrant">
													Entrant (Remise des clés)
												</Select.Item>
												<Select.Item value="MOVE_OUT" label="Sortant">
													Sortant (Comparatif)
												</Select.Item>
											</Select.Group>
										</Select.Content>
									</Select.Root>
								</Field.Field>

								<Field.Field>
									<Field.Label>Modèle de grille</Field.Label>
									<Select.Root
										type="single"
										value={newInspectionTemplateId}
										onValueChange={(v) => {
											if (v) newInspectionTemplateId = v;
										}}
									>
										<Select.Trigger class="w-full">
											{data.templates.find((t) => t.id === newInspectionTemplateId)?.name ??
												'Choisir un modèle'}
										</Select.Trigger>
										<Select.Content>
											<Select.Group>
												{#each data.templates as tpl (tpl.id)}
													<Select.Item value={tpl.id} label={tpl.name}>
														{tpl.name} ({tpl.rooms.length} pièces)
													</Select.Item>
												{/each}
											</Select.Group>
										</Select.Content>
									</Select.Root>
								</Field.Field>
							</div>

							{#if newInspectionType === 'MOVE_OUT'}
								<input type="hidden" name="entry_inspection_id" value={newInspectionEntryId} />
								<Field.Field>
									<Field.Label>
										État des lieux d'entrée de référence (comparatif automatique)
									</Field.Label>
									<Select.Root
										type="single"
										value={newInspectionEntryId}
										onValueChange={(v) => {
											if (v !== undefined) newInspectionEntryId = v;
										}}
									>
										<Select.Trigger class="w-full">
											{#if newInspectionEntryId}
												{@const entryInsp = moveInInspectionsForSelectedLease.find(
													(i) => i.id === newInspectionEntryId
												)}
												{entryInsp
													? `EDL Entrant du ${formatIsoDateFr(entryInsp.inspection_date)} (${entryInsp.id.slice(0, 8)})`
													: newInspectionEntryId}
											{:else}
												Aucun EDL d'entrée rattaché (rapport autonome)
											{/if}
										</Select.Trigger>
										<Select.Content>
											<Select.Group>
												<Select.Item value="" label="Aucun (rapport autonome)">
													Aucun EDL d'entrée rattaché (rapport autonome)
												</Select.Item>
												{#each moveInInspectionsForSelectedLease as entryInsp (entryInsp.id)}
													<Select.Item
														value={entryInsp.id}
														label={`EDL Entrant du ${formatIsoDateFr(entryInsp.inspection_date)}`}
													>
														EDL Entrant du {formatIsoDateFr(entryInsp.inspection_date)} ({entryInsp
															.items.length} éléments)
													</Select.Item>
												{/each}
											</Select.Group>
										</Select.Content>
									</Select.Root>
								</Field.Field>
							{/if}

							<div class="grid gap-3 sm:grid-cols-2">
								<Field.Field>
									<Field.Label for="keys-count">Nombre de clés remises</Field.Label>
									<Input
										id="keys-count"
										name="keys_count"
										type="number"
										min="0"
										bind:value={newInspectionKeysCount}
									/>
								</Field.Field>
								<Field.Field>
									<Field.Label for="keys-desc">Détail des clés / badges</Field.Label>
									<Input
										id="keys-desc"
										name="keys_description"
										bind:value={newInspectionKeysDesc}
									/>
								</Field.Field>
							</div>
						</Field.FieldGroup>

						<Button
							type="submit"
							disabled={isCreatingInspection ||
								data.leases.length === 0 ||
								data.templates.length === 0}
						>
							{#if isCreatingInspection}
								<Spinner data-icon="inline-start" />
							{/if}
							Ouvrir la grille d'état des lieux →
						</Button>
					</form>
				</Card.Content>
			</Card.Root>

			<!-- Templates & Session Inspections List -->
			<Card.Root>
				<Card.Header>
					<Card.Title class="text-sm font-semibold">
						Modèles disponibles & États des lieux récents
					</Card.Title>
					<Card.Description>
						Grilles types réutilisables par typologie de lot (Studio, T2, T3, Local commercial)
					</Card.Description>
				</Card.Header>
				<Card.Content class="flex flex-col gap-4">
					{#if data.templates.length === 0}
						<p class="text-xs text-muted-foreground">
							Aucun modèle configuré. Cliquez sur « Nouveau modèle EDL » en haut à droite.
						</p>
					{:else}
						<div class="flex flex-col gap-2">
							{#each data.templates as tpl (tpl.id)}
								<div class="flex items-center justify-between rounded-md border p-2.5 text-xs">
									<div class="flex flex-col gap-0.5">
										<span class="font-medium">{tpl.name}</span>
										<span class="text-[11px] text-muted-foreground">
											Pièces : {tpl.rooms.join(', ')}
										</span>
									</div>
									<Badge variant="outline">{tpl.rooms.length} pièces</Badge>
								</div>
							{/each}
						</div>
					{/if}

					{#if data.inspections.length > 0}
						<Separator />
						<div class="flex flex-col gap-2">
							<span class="text-xs font-semibold">
								États des lieux enregistrés ({data.inspections.length})
							</span>
							{#each data.inspections as insp (insp.id)}
								{@const lease = leaseMap.get(insp.lease_id)}
								<div
									class="flex items-center justify-between gap-2 rounded-md border p-2.5 text-xs"
								>
									<div class="flex min-w-0 flex-col">
										<span class="truncate font-medium">
											EDL {formatInspectionTypeLabel(insp.type)} — {formatIsoDateFr(
												insp.inspection_date
											)}
										</span>
										<span class="truncate text-[11px] text-muted-foreground">
											{lease ? formatLeaseDisplayLabel(lease) : insp.lease_id} · {insp.items.length} élément(s)
										</span>
									</div>
									<div class="flex shrink-0 items-center gap-1.5">
										{#if insp.report_download_url}
											<Button
												size="sm"
												variant="outline"
												href={insp.report_download_url}
												target="_blank"
												rel="noreferrer"
											>
												<IconDownload data-icon="inline-start" />
												PDF
											</Button>
										{/if}
										<Button size="sm" variant="secondary" onclick={() => openInspectionSheet(insp)}>
											Grille →
										</Button>
									</div>
								</div>
							{/each}
						</div>
					{/if}
				</Card.Content>
			</Card.Root>
		</div>
	</Tabs.Content>
</Tabs.Root>

<!-- Create Incident Sheet -->
<Sheet.Root bind:open={createIncidentOpen}>
	<Sheet.Content side="right" class="w-full overflow-y-auto sm:max-w-md">
		<Sheet.Header class="border-b pb-4">
			<Sheet.Title>Déclarer un incident ou sinistre</Sheet.Title>
			<Sheet.Description>
				Associez l'incident à un bien ou un bail pour qu'il remonte sur le Cockpit Opérationnel.
			</Sheet.Description>
		</Sheet.Header>

		<form
			method="POST"
			action="?/createIncident"
			use:enhance={() => {
				isCreatingIncident = true;
				return async ({ update }) => {
					await update();
					isCreatingIncident = false;
				};
			}}
			class="flex flex-col gap-4 p-4"
		>
			<input type="hidden" name="property_id" value={newIncidentPropertyId} />
			<input type="hidden" name="lease_id" value={newIncidentLeaseId} />

			<Field.FieldGroup>
				<Field.Field>
					<Field.Label for="inc-title">Intitulé de l'incident</Field.Label>
					<Input
						id="inc-title"
						name="title"
						placeholder="Dégât des eaux salle de bain"
						bind:value={newIncidentTitle}
						required
					/>
				</Field.Field>

				<Field.Field>
					<Field.Label>Bien immobilier concerné</Field.Label>
					<Select.Root
						type="single"
						value={newIncidentPropertyId}
						onValueChange={(v) => {
							if (v !== undefined) newIncidentPropertyId = v;
						}}
					>
						<Select.Trigger class="w-full">
							{propertyMap.get(newIncidentPropertyId)?.label ?? 'Sélectionner un bien'}
						</Select.Trigger>
						<Select.Content>
							<Select.Group>
								{#each data.properties as prop (prop.id)}
									<Select.Item value={prop.id} label={prop.label}>{prop.label}</Select.Item>
								{/each}
							</Select.Group>
						</Select.Content>
					</Select.Root>
				</Field.Field>

				<Field.Field>
					<Field.Label>Bail locatif (optionnel)</Field.Label>
					<Select.Root
						type="single"
						value={newIncidentLeaseId}
						onValueChange={(v) => {
							if (v !== undefined) newIncidentLeaseId = v;
						}}
					>
						<Select.Trigger class="w-full">
							{#if newIncidentLeaseId && leaseMap.get(newIncidentLeaseId)}
								{formatLeaseDisplayLabel(leaseMap.get(newIncidentLeaseId)!)}
							{:else}
								Aucun bail spécifique
							{/if}
						</Select.Trigger>
						<Select.Content>
							<Select.Group>
								{#each data.leases as lease (lease.id)}
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
						<Field.Label for="inc-claim">N° dossier assurance</Field.Label>
						<Input
							id="inc-claim"
							name="insurance_claim_number"
							placeholder="SIN-2026-884"
							bind:value={newIncidentClaimNumber}
						/>
					</Field.Field>

					<Field.Field>
						<Field.Label for="inc-cost">Coût estimé (€)</Field.Label>
						<Input
							id="inc-cost"
							name="estimated_cost_euro"
							type="text"
							inputmode="decimal"
							placeholder="350,00"
							bind:value={newIncidentEstimatedEuro}
						/>
					</Field.Field>
				</div>

				<Field.Field>
					<Field.Label for="inc-obs">Observation initiale</Field.Label>
					<Input
						id="inc-obs"
						name="observation"
						placeholder="Plombier mandaté pour recherche de fuite"
						bind:value={newIncidentObservation}
					/>
				</Field.Field>
			</Field.FieldGroup>

			<Button type="submit" disabled={isCreatingIncident}>
				{#if isCreatingIncident}
					<Spinner data-icon="inline-start" />
				{/if}
				Enregistrer l'incident
			</Button>
		</form>
	</Sheet.Content>
</Sheet.Root>

<!-- Edit Incident Sheet -->
<Sheet.Root bind:open={editIncidentOpen}>
	<Sheet.Content side="right" class="w-full overflow-y-auto sm:max-w-md">
		<Sheet.Header class="border-b pb-4">
			<Sheet.Title>{selectedIncident?.title ?? 'Suivi incident'}</Sheet.Title>
			<Sheet.Description>
				Déclaré le {formatIsoDateFr(selectedIncident?.occurred_at)}
			</Sheet.Description>
		</Sheet.Header>

		{#if selectedIncident}
			<form
				method="POST"
				action="?/updateIncident"
				use:enhance={() => {
					isUpdatingIncident = true;
					return async ({ update }) => {
						await update();
						isUpdatingIncident = false;
					};
				}}
				class="flex flex-col gap-4 p-4"
			>
				<input type="hidden" name="incident_id" value={selectedIncident.id} />
				<input type="hidden" name="status" value={editIncidentStatus} />

				<Field.FieldGroup>
					<Field.Field>
						<Field.Label>Statut d'avancement</Field.Label>
						<Select.Root
							type="single"
							value={editIncidentStatus}
							onValueChange={(v) => {
								if (v) editIncidentStatus = v;
							}}
						>
							<Select.Trigger class="w-full">
								{formatIncidentStatusLabel(editIncidentStatus)}
							</Select.Trigger>
							<Select.Content>
								<Select.Group>
									{#each INCIDENT_STATUS_OPTIONS as opt (opt.value)}
										<Select.Item value={opt.value} label={opt.label}>{opt.label}</Select.Item>
									{/each}
								</Select.Group>
							</Select.Content>
						</Select.Root>
					</Field.Field>

					<div class="grid gap-3 sm:grid-cols-2">
						<Field.Field>
							<Field.Label for="edit-inc-claim">N° sinistre assurance</Field.Label>
							<Input
								id="edit-inc-claim"
								name="insurance_claim_number"
								bind:value={editIncidentClaimNumber}
							/>
						</Field.Field>

						<Field.Field>
							<Field.Label for="edit-inc-cost">Coût estimé (€)</Field.Label>
							<Input
								id="edit-inc-cost"
								name="estimated_cost_euro"
								type="text"
								inputmode="decimal"
								bind:value={editIncidentCostEuro}
							/>
						</Field.Field>
					</div>

					<Field.Field>
						<Field.Label for="edit-inc-obs">Ajouter une note de suivi</Field.Label>
						<Input
							id="edit-inc-obs"
							name="add_observation"
							placeholder="Devis validé / intervention planifiée..."
							bind:value={editIncidentNewObservation}
						/>
					</Field.Field>
				</Field.FieldGroup>

				<Button type="submit" disabled={isUpdatingIncident}>
					{#if isUpdatingIncident}
						<Spinner data-icon="inline-start" />
					{/if}
					Enregistrer les modifications
				</Button>

				{#if selectedIncident.observations && selectedIncident.observations.length > 0}
					<Separator />
					<div class="flex flex-col gap-2">
						<span class="text-xs font-semibold uppercase">Historique des observations</span>
						{#each selectedIncident.observations as obs, index (`${selectedIncident.id}-obs-${index}`)}
							<div class="rounded-md border bg-muted/40 p-2.5 text-xs">
								{obs}
							</div>
						{/each}
					</div>
				{/if}
			</form>
		{/if}
	</Sheet.Content>
</Sheet.Root>

<!-- Inspection Room-by-Room Sheet -->
<Sheet.Root bind:open={inspectionSheetOpen}>
	<Sheet.Content side="right" class="w-full overflow-y-auto sm:max-w-xl">
		<Sheet.Header class="border-b pb-4">
			<Sheet.Title>
				Grille d'état des lieux {activeInspection
					? formatInspectionTypeLabel(activeInspection.type)
					: ''}
			</Sheet.Title>
			<Sheet.Description>
				Qualifiez chaque équipement pièce par pièce puis générez le rapport PDF contradictoire.
			</Sheet.Description>
		</Sheet.Header>

		{#if activeInspection}
			<div class="flex flex-col gap-4 p-4">
				<div class="flex flex-col gap-2.5">
					{#each activeInspectionItems as item, idx (`${item.room_name}-${item.element_name}-${idx}`)}
						<div class="grid gap-2 rounded-md border p-2.5 sm:grid-cols-12 sm:items-center">
							<div class="sm:col-span-4">
								<p class="text-xs font-semibold">{item.room_name}</p>
								<p class="text-[11px] text-muted-foreground">{item.element_name}</p>
							</div>
							<div class="sm:col-span-4">
								<Select.Root
									type="single"
									value={item.condition}
									onValueChange={(val) => {
										if (val) activeInspectionItems[idx].condition = val;
									}}
								>
									<Select.Trigger class="w-full">
										{CONDITION_OPTIONS.find((c) => c.value === item.condition)?.label ??
											item.condition}
									</Select.Trigger>
									<Select.Content>
										<Select.Group>
											{#each CONDITION_OPTIONS as cond (cond.value)}
												<Select.Item value={cond.value} label={cond.label}>
													{cond.label}
												</Select.Item>
											{/each}
										</Select.Group>
									</Select.Content>
								</Select.Root>
							</div>
							<div class="sm:col-span-4">
								<Input
									placeholder="Observation..."
									value={item.observation ?? ''}
									oninput={(e) => {
										activeInspectionItems[idx].observation = (
											e.currentTarget as HTMLInputElement
										).value;
									}}
								/>
							</div>
						</div>
					{/each}
				</div>

				<div class="flex flex-wrap items-center gap-2">
					<form
						method="POST"
						action="?/saveInspectionItems"
						use:enhance={() => {
							isSavingInspectionItems = true;
							return async ({ update }) => {
								await update();
								isSavingInspectionItems = false;
							};
						}}
					>
						<input type="hidden" name="inspection_id" value={activeInspection.id} />
						<input type="hidden" name="items_json" value={JSON.stringify(activeInspectionItems)} />
						<Button type="submit" variant="secondary" disabled={isSavingInspectionItems}>
							{#if isSavingInspectionItems}
								<Spinner data-icon="inline-start" />
							{/if}
							Sauvegarder la grille
						</Button>
					</form>

					<form
						method="POST"
						action="?/generatePdf"
						use:enhance={() => {
							isGeneratingPdf = true;
							return async ({ update }) => {
								await update();
								isGeneratingPdf = false;
							};
						}}
					>
						<input type="hidden" name="inspection_id" value={activeInspection.id} />
						<input type="hidden" name="items_json" value={JSON.stringify(activeInspectionItems)} />
						<Button type="submit" disabled={isGeneratingPdf}>
							{#if isGeneratingPdf}
								<Spinner data-icon="inline-start" />
							{:else}
								<IconFileText data-icon="inline-start" />
							{/if}
							Générer le PDF contradictoire
						</Button>
					</form>
				</div>

				{#if activeInspectionReport}
					<div class="flex flex-col gap-2 rounded-md border bg-muted/40 p-3 text-xs">
						<div class="flex items-center justify-between">
							<span class="font-semibold">Rapport PDF généré</span>
							{#if activeInspectionReport.download_url}
								<Button
									size="sm"
									variant="outline"
									href={activeInspectionReport.download_url}
									target="_blank"
									rel="noreferrer"
								>
									<IconDownload data-icon="inline-start" />
									Télécharger le PDF
								</Button>
							{/if}
						</div>

						{#if activeInspectionReport.degradations && activeInspectionReport.degradations.length > 0}
							<p class="font-medium text-destructive">
								{activeInspectionReport.degradations.length} dégradation(s) retenue(s) entre l'entrée
								et la sortie.
							</p>
						{/if}
					</div>
				{/if}
			</div>
		{/if}
	</Sheet.Content>
</Sheet.Root>

<!-- Create Inspection Template Dialog -->
<Dialog.Root bind:open={createTemplateOpen}>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>Créer un modèle d'état des lieux</Dialog.Title>
			<Dialog.Description>
				Définissez la liste des pièces et des éléments inspectés par défaut.
			</Dialog.Description>
		</Dialog.Header>

		<form
			method="POST"
			action="?/createTemplate"
			use:enhance={() => {
				isCreatingTemplate = true;
				return async ({ update }) => {
					await update();
					isCreatingTemplate = false;
				};
			}}
			class="flex flex-col gap-4"
		>
			<Field.FieldGroup>
				<Field.Field>
					<Field.Label for="tpl-name">Nom du modèle</Field.Label>
					<Input
						id="tpl-name"
						name="name"
						placeholder="Appartement T2 Standard"
						bind:value={newTemplateName}
						required
					/>
				</Field.Field>

				<Field.Field>
					<Field.Label for="tpl-rooms">Pièces (séparées par des virgules)</Field.Label>
					<Input id="tpl-rooms" name="rooms_text" bind:value={newTemplateRoomsText} required />
				</Field.Field>

				<Field.Field>
					<Field.Label for="tpl-elems">Éléments par pièce (séparés par des virgules)</Field.Label>
					<Input
						id="tpl-elems"
						name="elements_text"
						bind:value={newTemplateElementsText}
						required
					/>
				</Field.Field>
			</Field.FieldGroup>

			<Dialog.Footer>
				<Button variant="outline" type="button" onclick={() => (createTemplateOpen = false)}>
					Annuler
				</Button>
				<Button type="submit" disabled={isCreatingTemplate}>
					{#if isCreatingTemplate}
						<Spinner data-icon="inline-start" />
					{/if}
					Créer le modèle
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
