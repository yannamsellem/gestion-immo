<script lang="ts">
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
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import * as Table from '#lib/components/ui/table/index.js';
	import * as Tabs from '#lib/components/ui/tabs/index.js';
	import { ApiError } from '#lib/core/api/api-error';
	import { formatIsoDateFr } from '#lib/core/formatters/date';
	import { formatCentsToEuro, parseEuroToCents } from '#lib/core/formatters/money';
	import { workspaceStore } from '#lib/core/workspace/workspace.svelte';
	import type {
		ConditionState,
		IncidentDTO,
		IncidentStatus,
		InspectionDTO,
		InspectionItemInput,
		InspectionReportDTO,
		InspectionTemplateDTO,
		InspectionType
	} from '#lib/modules/inspection/types';
	import type { LeaseDTO, TenantDTO } from '#lib/modules/lease/types';
	import type { PropertyDTO } from '#lib/modules/portfolio/types';
	import { apiRegistry } from '#lib/modules/registry';

	const INCIDENT_STATUS_OPTIONS: Array<{ value: IncidentStatus; label: string }> = [
		{ value: 'OPEN', label: 'Ouvert' },
		{ value: 'IN_PROGRESS', label: 'En cours' },
		{ value: 'RESOLVED', label: 'Résolu' },
		{ value: 'CLOSED', label: 'Clôturé' }
	];

	const CONDITION_OPTIONS: Array<{ value: ConditionState; label: string }> = [
		{ value: 'NEW', label: 'Neuf' },
		{ value: 'GOOD', label: 'Bon état' },
		{ value: 'FAIR', label: 'État d’usage' },
		{ value: 'POOR', label: 'Mauvais état' },
		{ value: 'DAMAGED', label: 'Dégradé' }
	];

	let activeTab = $state<'incidents' | 'inspections'>('incidents');
	let isLoading = $state(true);
	let errorMessage = $state<string | null>(null);
	let reloadCounter = $state(0);

	let incidents = $state<IncidentDTO[]>([]);
	let templates = $state<InspectionTemplateDTO[]>([]);
	let properties = $state<PropertyDTO[]>([]);
	let leases = $state<LeaseDTO[]>([]);
	let tenants = $state<TenantDTO[]>([]);

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
	let newInspectionType = $state<InspectionType>('ENTRANT');
	let newInspectionEntryId = $state('');
	let newInspectionKeysCount = $state('2');
	let newInspectionKeysDesc = $state('Clés porte blindée + badge immeuble');
	let isCreatingInspection = $state(false);

	let recentInspections = $state<InspectionDTO[]>([]);
	let activeInspection = $state<InspectionDTO | null>(null);
	let activeInspectionItems = $state<InspectionItemInput[]>([]);
	let activeInspectionReport = $state<InspectionReportDTO | null>(null);
	let inspectionSheetOpen = $state(false);
	let isSavingInspectionItems = $state(false);
	let isGeneratingPdf = $state(false);

	$effect(() => {
		const urlEntityId = page.url.searchParams.get('entityId');
		if (urlEntityId && urlEntityId !== workspaceStore.selectedEntityId) {
			workspaceStore.setEntityId(urlEntityId);
		}
		const urlTab = page.url.searchParams.get('tab');
		if (urlTab === 'inspections' || urlTab === 'incidents') {
			activeTab = urlTab;
		}
	});

	$effect(() => {
		const selectedEntityId = workspaceStore.selectedEntityId;
		const trigger = reloadCounter;
		void trigger;

		let cancelled = false;
		isLoading = true;
		errorMessage = null;

		void (async () => {
			try {
				const allEntities = await apiRegistry.portfolio.listEntities();
				if (cancelled) return;
				workspaceStore.setEntities(allEntities);

				const targetEntities =
					selectedEntityId === 'ALL'
						? allEntities
						: allEntities.filter((e) => e.id === selectedEntityId);

				const [loadedIncidents, loadedTemplates, loadedLeases, loadedTenants, propArrays] =
					await Promise.all([
						apiRegistry.inspection.listIncidents(),
						apiRegistry.inspection.listTemplates(),
						apiRegistry.lease.listLeases(selectedEntityId),
						apiRegistry.lease.listTenants(true),
						Promise.all(
							targetEntities.map((entity) => apiRegistry.portfolio.listProperties(entity.id))
						)
					]);

				if (cancelled) return;

				const flatProps = propArrays.flat();
				const allowedPropertyIds = new Set(flatProps.map((p) => p.id));
				const allowedLeaseIds = new Set(loadedLeases.map((l) => l.id));

				incidents =
					selectedEntityId === 'ALL'
						? loadedIncidents
						: loadedIncidents.filter(
								(inc) =>
									(inc.property_id && allowedPropertyIds.has(inc.property_id)) ||
									(inc.lease_id && allowedLeaseIds.has(inc.lease_id)) ||
									(!inc.property_id && !inc.lease_id)
							);
				templates = loadedTemplates;
				properties = flatProps;
				leases = loadedLeases.filter((l) => l.active);
				tenants = loadedTenants;

				if (flatProps.length > 0 && !newIncidentPropertyId) {
					newIncidentPropertyId = flatProps[0].id;
				}
				if (leases.length > 0 && !newInspectionLeaseId) {
					newInspectionLeaseId = leases[0].id;
				}
				if (loadedTemplates.length > 0 && !newInspectionTemplateId) {
					newInspectionTemplateId = loadedTemplates[0].id;
				}
			} catch (err) {
				if (cancelled) return;
				errorMessage =
					err instanceof ApiError
						? err.message
						: 'Impossible de charger les incidents et états des lieux.';
			} finally {
				if (!cancelled) {
					isLoading = false;
				}
			}
		})();

		return () => {
			cancelled = true;
		};
	});

	// Auto-open incident Sheet when arriving from Cockpit with ?incidentId=...
	$effect(() => {
		if (isLoading || incidents.length === 0) return;
		const urlIncidentId = page.url.searchParams.get('incidentId');
		if (!urlIncidentId || autoOpenedIncidentId === urlIncidentId) return;

		const match = incidents.find((inc) => inc.id === urlIncidentId);
		if (match) {
			openEditIncidentSheet(match);
			autoOpenedIncidentId = urlIncidentId;
		}
	});

	const propertyMap = $derived(new Map(properties.map((p) => [p.id, p])));
	const tenantMap = $derived(new Map(tenants.map((t) => [t.id, t])));
	const leaseMap = $derived(new Map(leases.map((l) => [l.id, l])));

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

	async function handleCreateIncident(event: SubmitEvent) {
		event.preventDefault();
		if (!newIncidentTitle.trim()) return;

		let estimatedCostCents = 0;
		try {
			estimatedCostCents = parseEuroToCents(newIncidentEstimatedEuro || '0');
		} catch {
			toast.error('Montant estimé invalide.');
			return;
		}

		isCreatingIncident = true;
		try {
			await apiRegistry.inspection.createIncident({
				property_id: newIncidentPropertyId || undefined,
				lease_id: newIncidentLeaseId || undefined,
				title: newIncidentTitle.trim(),
				status: 'OPEN',
				insurance_claim_number: newIncidentClaimNumber.trim() || undefined,
				estimated_cost_cents: estimatedCostCents,
				observations: newIncidentObservation.trim() ? [newIncidentObservation.trim()] : []
			});
			toast.success('Incident enregistré.');
			createIncidentOpen = false;
			newIncidentTitle = '';
			newIncidentClaimNumber = '';
			newIncidentObservation = '';
			reloadCounter += 1;
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Erreur lors de la création.');
		} finally {
			isCreatingIncident = false;
		}
	}

	async function handleUpdateIncident(event: SubmitEvent) {
		event.preventDefault();
		if (!selectedIncident) return;

		let estimatedCostCents = 0;
		try {
			estimatedCostCents = parseEuroToCents(editIncidentCostEuro || '0');
		} catch {
			toast.error('Coût estimé invalide.');
			return;
		}

		isUpdatingIncident = true;
		try {
			const updated = await apiRegistry.inspection.updateIncident(selectedIncident.id, {
				status: editIncidentStatus,
				insurance_claim_number: editIncidentClaimNumber.trim() || undefined,
				estimated_cost_cents: estimatedCostCents,
				add_observation: editIncidentNewObservation.trim() || undefined
			});
			selectedIncident = updated;
			editIncidentNewObservation = '';
			toast.success('Incident mis à jour.');
			reloadCounter += 1;
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Impossible de mettre à jour l’incident.');
		} finally {
			isUpdatingIncident = false;
		}
	}

	async function handleCreateTemplate(event: SubmitEvent) {
		event.preventDefault();
		const rooms = newTemplateRoomsText
			.split(',')
			.map((s) => s.trim())
			.filter(Boolean);
		const defaultElements = newTemplateElementsText
			.split(',')
			.map((s) => s.trim())
			.filter(Boolean);

		if (!newTemplateName.trim() || rooms.length === 0) {
			toast.error('Veuillez renseigner un nom et au moins une pièce.');
			return;
		}

		isCreatingTemplate = true;
		try {
			const created = await apiRegistry.inspection.createTemplate({
				name: newTemplateName.trim(),
				rooms,
				default_elements: defaultElements
			});
			templates = [created, ...templates];
			newInspectionTemplateId = created.id;
			createTemplateOpen = false;
			newTemplateName = '';
			toast.success('Modèle d’état des lieux créé.');
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Erreur lors de la création du modèle.');
		} finally {
			isCreatingTemplate = false;
		}
	}

	async function handleCreateInspection(event: SubmitEvent) {
		event.preventDefault();
		if (!newInspectionLeaseId || !newInspectionTemplateId) {
			toast.error('Sélectionnez un bail et un modèle d’état des lieux.');
			return;
		}

		isCreatingInspection = true;
		try {
			const created = await apiRegistry.inspection.createInspection({
				lease_id: newInspectionLeaseId,
				template_id: newInspectionTemplateId,
				type: newInspectionType,
				entry_inspection_id:
					newInspectionType === 'SORTANT' && newInspectionEntryId.trim()
						? newInspectionEntryId.trim()
						: undefined,
				keys_handed_over_count: Number.parseInt(newInspectionKeysCount, 10) || 0,
				keys_description: newInspectionKeysDesc.trim() || undefined
			});
			recentInspections = [created, ...recentInspections];
			openInspectionSheet(created);
			toast.success(`État des lieux ${created.type} initialisé.`);
		} catch (err) {
			toast.error(
				err instanceof ApiError ? err.message : 'Impossible de créer l’état des lieux.'
			);
		} finally {
			isCreatingInspection = false;
		}
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
			const tpl = templates.find((t) => t.id === inspection.template_id);
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

	async function handleSaveInspectionItems() {
		if (!activeInspection) return;
		isSavingInspectionItems = true;
		try {
			const updated = await apiRegistry.inspection.upsertInspectionItems(
				activeInspection.id,
				activeInspectionItems
			);
			activeInspection = updated;
			recentInspections = recentInspections.map((item) =>
				item.id === updated.id ? updated : item
			);
			toast.success('Relevé pièce par pièce enregistré.');
		} catch (err) {
			toast.error(
				err instanceof ApiError ? err.message : 'Impossible d’enregistrer les éléments.'
			);
		} finally {
			isSavingInspectionItems = false;
		}
	}

	async function handleGenerateInspectionPdf() {
		if (!activeInspection) return;
		isGeneratingPdf = true;
		try {
			await handleSaveInspectionItems();
			const report = await apiRegistry.inspection.generateInspectionPdf(activeInspection.id);
			activeInspectionReport = report;
			toast.success('Rapport PDF contradictoire généré.');
		} catch (err) {
			toast.error(
				err instanceof ApiError ? err.message : 'Échec de la génération du rapport PDF.'
			);
		} finally {
			isGeneratingPdf = false;
		}
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

{#if errorMessage}
	<Alert.Root variant="destructive">
		<IconAlertTriangle class="size-4" />
		<Alert.Title>Erreur</Alert.Title>
		<Alert.Description>{errorMessage}</Alert.Description>
	</Alert.Root>
{/if}

<Tabs.Root bind:value={activeTab}>
	<Tabs.List>
		<Tabs.Trigger value="incidents">
			Incidents & Sinistres ({incidents.length})
		</Tabs.Trigger>
		<Tabs.Trigger value="inspections">
			États des lieux & Modèles ({templates.length})
		</Tabs.Trigger>
	</Tabs.List>

	<!-- Tab 1: Incidents -->
	<Tabs.Content value="incidents" class="mt-4">
		{#if isLoading}
			<Skeleton class="h-48 w-full" />
		{:else if incidents.length === 0}
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
							{#each incidents as incident (incident.id)}
								<Table.Row
									class="cursor-pointer"
									onclick={() => openEditIncidentSheet(incident)}
								>
									<Table.Cell class="pl-4 text-xs">
										{formatIsoDateFr(incident.occurred_at)}
									</Table.Cell>
									<Table.Cell class="font-medium">{incident.title}</Table.Cell>
									<Table.Cell class="text-muted-foreground text-xs">
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
					<form onsubmit={handleCreateInspection} class="flex flex-col gap-4">
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
											{#each leases as lease (lease.id)}
												<Select.Item
													value={lease.id}
													label={formatLeaseDisplayLabel(lease)}
												>
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
											{newInspectionType === 'ENTRANT'
												? 'Entrant (Remise des clés)'
												: 'Sortant (Comparatif)'}
										</Select.Trigger>
										<Select.Content>
											<Select.Group>
												<Select.Item value="ENTRANT" label="Entrant">
													Entrant (Remise des clés)
												</Select.Item>
												<Select.Item value="SORTANT" label="Sortant">
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
											{templates.find((t) => t.id === newInspectionTemplateId)?.name ??
												'Choisir un modèle'}
										</Select.Trigger>
										<Select.Content>
											<Select.Group>
												{#each templates as tpl (tpl.id)}
													<Select.Item value={tpl.id} label={tpl.name}>
														{tpl.name} ({tpl.rooms.length} pièces)
													</Select.Item>
												{/each}
											</Select.Group>
										</Select.Content>
									</Select.Root>
								</Field.Field>
							</div>

							{#if newInspectionType === 'SORTANT'}
								<Field.Field>
									<Field.Label for="entry-edl-id">
										ID de l'état des lieux d'entrée (pour comparatif automatique)
									</Field.Label>
									<Input
										id="entry-edl-id"
										placeholder="UUID de l'EDL entrant (optionnel)"
										bind:value={newInspectionEntryId}
									/>
								</Field.Field>
							{/if}

							<div class="grid gap-3 sm:grid-cols-2">
								<Field.Field>
									<Field.Label for="keys-count">Nombre de clés remises</Field.Label>
									<Input
										id="keys-count"
										type="number"
										min="0"
										bind:value={newInspectionKeysCount}
									/>
								</Field.Field>
								<Field.Field>
									<Field.Label for="keys-desc">Détail des clés / badges</Field.Label>
									<Input id="keys-desc" bind:value={newInspectionKeysDesc} />
								</Field.Field>
							</div>
						</Field.FieldGroup>

						<Button
							type="submit"
							disabled={isCreatingInspection || leases.length === 0 || templates.length === 0}
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
					{#if templates.length === 0}
						<p class="text-muted-foreground text-xs">
							Aucun modèle configuré. Cliquez sur « Nouveau modèle EDL » en haut à droite.
						</p>
					{:else}
						<div class="flex flex-col gap-2">
							{#each templates as tpl (tpl.id)}
								<div class="flex items-center justify-between rounded-md border p-2.5 text-xs">
									<div class="flex flex-col gap-0.5">
										<span class="font-medium">{tpl.name}</span>
										<span class="text-muted-foreground text-[11px]">
											Pièces : {tpl.rooms.join(', ')}
										</span>
									</div>
									<Badge variant="outline">{tpl.rooms.length} pièces</Badge>
								</div>
							{/each}
						</div>
					{/if}

					{#if recentInspections.length > 0}
						<Separator />
						<div class="flex flex-col gap-2">
							<span class="text-xs font-semibold">États des lieux ouverts dans cette session</span>
							{#each recentInspections as insp (insp.id)}
								<div class="flex items-center justify-between rounded-md border p-2.5 text-xs">
									<div class="flex flex-col">
										<span class="font-medium">
											EDL {insp.type} — {formatIsoDateFr(insp.inspection_date)}
										</span>
										<span class="text-muted-foreground font-mono text-[11px]">
											ID : {insp.id}
										</span>
									</div>
									<Button
										size="sm"
										variant="secondary"
										onclick={() => openInspectionSheet(insp)}
									>
										Compléter / PDF →
									</Button>
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

		<form onsubmit={handleCreateIncident} class="flex flex-col gap-4 p-4">
			<Field.FieldGroup>
				<Field.Field>
					<Field.Label for="inc-title">Intitulé de l'incident</Field.Label>
					<Input
						id="inc-title"
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
								{#each properties as prop (prop.id)}
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
						<Field.Label for="inc-claim">N° dossier assurance</Field.Label>
						<Input
							id="inc-claim"
							placeholder="SIN-2026-884"
							bind:value={newIncidentClaimNumber}
						/>
					</Field.Field>

					<Field.Field>
						<Field.Label for="inc-cost">Coût estimé (€)</Field.Label>
						<Input
							id="inc-cost"
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
			<form onsubmit={handleUpdateIncident} class="flex flex-col gap-4 p-4">
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
							<Input id="edit-inc-claim" bind:value={editIncidentClaimNumber} />
						</Field.Field>

						<Field.Field>
							<Field.Label for="edit-inc-cost">Coût estimé (€)</Field.Label>
							<Input
								id="edit-inc-cost"
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
							<div class="bg-muted/40 rounded-md border p-2.5 text-xs">
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
				Grille d'état des lieux {activeInspection?.type ?? ''}
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
								<p class="text-muted-foreground text-[11px]">{item.element_name}</p>
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
					<Button
						variant="secondary"
						onclick={handleSaveInspectionItems}
						disabled={isSavingInspectionItems}
					>
						{#if isSavingInspectionItems}
							<Spinner data-icon="inline-start" />
						{/if}
						Sauvegarder la grille
					</Button>

					<Button onclick={handleGenerateInspectionPdf} disabled={isGeneratingPdf}>
						{#if isGeneratingPdf}
							<Spinner data-icon="inline-start" />
						{:else}
							<IconFileText data-icon="inline-start" />
						{/if}
						Générer le PDF contradictoire
					</Button>
				</div>

				{#if activeInspectionReport}
					<div class="bg-muted/40 flex flex-col gap-2 rounded-md border p-3 text-xs">
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
							<p class="text-destructive font-medium">
								{activeInspectionReport.degradations.length} dégradation(s) retenue(s) entre
								l'entrée et la sortie.
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

		<form onsubmit={handleCreateTemplate} class="flex flex-col gap-4">
			<Field.FieldGroup>
				<Field.Field>
					<Field.Label for="tpl-name">Nom du modèle</Field.Label>
					<Input
						id="tpl-name"
						placeholder="Appartement T2 Standard"
						bind:value={newTemplateName}
						required
					/>
				</Field.Field>

				<Field.Field>
					<Field.Label for="tpl-rooms">Pièces (séparées par des virgules)</Field.Label>
					<Input id="tpl-rooms" bind:value={newTemplateRoomsText} required />
				</Field.Field>

				<Field.Field>
					<Field.Label for="tpl-elems">Éléments par pièce (séparés par des virgules)</Field.Label>
					<Input id="tpl-elems" bind:value={newTemplateElementsText} required />
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
