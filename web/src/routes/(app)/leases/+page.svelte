<script lang="ts">
	import { page } from '$app/state';
	import IconAlertTriangle from '@tabler/icons-svelte/icons/alert-triangle';
	import IconArchive from '@tabler/icons-svelte/icons/archive';
	import IconDownload from '@tabler/icons-svelte/icons/download';
	import IconFileDescription from '@tabler/icons-svelte/icons/file-description';
	import IconFileUpload from '@tabler/icons-svelte/icons/file-upload';
	import IconPlus from '@tabler/icons-svelte/icons/plus';
	import IconUserPlus from '@tabler/icons-svelte/icons/user-plus';
	import { toast } from 'svelte-sonner';
	import PageHeader from '#lib/components/layout/PageHeader.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { Checkbox } from '#lib/components/ui/checkbox/index.js';
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
	import {
		formatBasisPointsToPercent,
		formatCentsToEuro,
		parseEuroToCents,
		parsePercentToBasisPoints
	} from '#lib/core/formatters/money';
	import { workspaceStore } from '#lib/core/workspace/workspace.svelte';
	import type {
		DocumentTemplateDTO,
		StoredFileDTO,
		TemplateFormat,
		TemplateType
	} from '#lib/modules/document/types';
	import type { LeaseDTO, LeaseType, TenantDTO } from '#lib/modules/lease/types';
	import type { LegalEntityDTO, LotDTO, PropertyDTO } from '#lib/modules/portfolio/types';
	import { apiRegistry } from '#lib/modules/registry';

	const LEASE_TYPES: Array<{ value: LeaseType; label: string }> = [
		{ value: 'RESIDENTIAL_EMPTY', label: 'Habitation vide (Loi 89 — 3 ans)' },
		{ value: 'RESIDENTIAL_FURNISHED', label: 'Habitation meublée (1 an / 9 mois)' },
		{ value: 'MOBILITY', label: 'Bail mobilité (1 à 10 mois)' },
		{ value: 'COMMERCIAL_3_6_9', label: 'Bail commercial 3 / 6 / 9' }
	];

	const TEMPLATE_TYPES: Array<{ value: TemplateType; label: string }> = [
		{ value: 'LEASE_CONTRACT', label: 'Contrat de bail' },
		{ value: 'RENT_CALL', label: 'Avis d’échéance' },
		{ value: 'RECEIPT', label: 'Quittance de loyer' },
		{ value: 'REMINDER', label: 'Lettre de relance impayé' },
		{ value: 'GLI_NOTICE', label: 'Mise en demeure / Déclaration GLI' }
	];

	let activeTab = $state<'leases' | 'tenants' | 'templates'>('leases');
	let isLoading = $state(true);
	let errorMessage = $state<string | null>(null);
	let reloadCounter = $state(0);

	let entities = $state<LegalEntityDTO[]>([]);
	let properties = $state<PropertyDTO[]>([]);
	let leases = $state<LeaseDTO[]>([]);
	let tenants = $state<TenantDTO[]>([]);
	let templates = $state<DocumentTemplateDTO[]>([]);

	// Create Lease Sheet
	let createLeaseOpen = $state(false);
	let leaseEntityId = $state('');
	let leaseLotId = $state('');
	let leaseTenantId = $state('');
	let leaseType = $state<LeaseType>('RESIDENTIAL_EMPTY');
	let leaseRentNetEuro = $state('800,00');
	let leaseChargesEuro = $state('75,00');
	let leaseSubjectToVat = $state(false);
	let leaseVatPercent = $state('20,00');
	let leaseDepositEuro = $state('800,00');
	let leaseDueDay = $state('5');
	let leaseGliPolicy = $state('');
	let leaseGuarantor = $state('');
	let leaseStartDate = $state(new Date().toISOString().slice(0, 10));
	let isCreatingLease = $state(false);

	// Create Tenant Sheet
	let createTenantOpen = $state(false);
	let tenantFullName = $state('');
	let tenantEmail = $state('');
	let tenantPhone = $state('');
	let tenantAddress = $state('');
	let isCreatingTenant = $state(false);

	// Lease Detail & Documents Sheet
	let leaseDocsSheetOpen = $state(false);
	let selectedLease = $state<LeaseDTO | null>(null);
	let selectedLeaseDocs = $state<StoredFileDTO[]>([]);
	let selectedTemplateId = $state('');
	let uploadCategory = $state('BAIL_SIGNE');
	let uploadFile = $state<File | null>(null);
	let isUploadingDoc = $state(false);
	let isGeneratingDoc = $state(false);

	// Register Template Dialog
	let registerTemplateOpen = $state(false);
	let tplName = $state('');
	let tplType = $state<TemplateType>('LEASE_CONTRACT');
	let tplFormat = $state<TemplateFormat>('HTML');
	let tplHtmlBody = $state(
		'<h1>{{entity_name}}</h1><p>Locataire : {{tenant_name}}</p><p>Lot : {{property_label}}</p>'
	);
	let tplDocxBase64 = $state('');
	let isRegisteringTemplate = $state(false);

	let autoHandledUrl = $state(false);

	$effect(() => {
		const urlEntityId = page.url.searchParams.get('entityId');
		if (urlEntityId && urlEntityId !== workspaceStore.selectedEntityId) {
			workspaceStore.setEntityId(urlEntityId);
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
				entities = allEntities;
				workspaceStore.setEntities(allEntities);

				const targetEntities =
					selectedEntityId === 'ALL'
						? allEntities
						: allEntities.filter((e) => e.id === selectedEntityId);

				if (targetEntities.length > 0 && !leaseEntityId) {
					leaseEntityId = targetEntities[0].id;
				}

				const [propLists, loadedLeases, loadedTenants, loadedTemplates] = await Promise.all([
					Promise.all(
						targetEntities.map((entity) => apiRegistry.portfolio.listProperties(entity.id))
					),
					apiRegistry.lease.listLeases(selectedEntityId),
					apiRegistry.lease.listTenants(true),
					apiRegistry.document.listTemplates().catch(() => [])
				]);
				if (cancelled) return;

				const flatProps = propLists.flat();
				properties = flatProps;
				leases = loadedLeases;
				tenants = loadedTenants;
				templates = loadedTemplates;

				const allLots = flatProps.flatMap((p) => p.lots ?? []);
				if (allLots.length > 0 && !leaseLotId) {
					leaseLotId = allLots[0].id;
				}
				const activeTenants = loadedTenants.filter((t) => !t.archived);
				if (activeTenants.length > 0 && !leaseTenantId) {
					leaseTenantId = activeTenants[0].id;
				}
				if (loadedTemplates.length > 0 && !selectedTemplateId) {
					selectedTemplateId = loadedTemplates[0].id;
				}

				if (!autoHandledUrl) {
					const action = page.url.searchParams.get('action');
					const urlLotId = page.url.searchParams.get('lotId');
					if (urlLotId) {
						leaseLotId = urlLotId;
						const ownerProp = flatProps.find((p) =>
							(p.lots ?? []).some((lot) => lot.id === urlLotId)
						);
						if (ownerProp) {
							leaseEntityId = ownerProp.legal_entity_id;
						}
					}
					if (action === 'create' || urlLotId) {
						createLeaseOpen = true;
						autoHandledUrl = true;
					}
				}
			} catch (err) {
				if (cancelled) return;
				errorMessage =
					err instanceof ApiError
						? err.message
						: 'Impossible de charger les baux et locataires.';
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

	const entityMap = $derived(new Map(entities.map((e) => [e.id, e])));
	const tenantMap = $derived(new Map(tenants.map((t) => [t.id, t])));
	const lotInfoMap = $derived.by(() => {
		const map = new Map<string, { lot: LotDTO; property: PropertyDTO }>();
		for (const prop of properties) {
			for (const lot of prop.lots ?? []) {
				map.set(lot.id, { lot, property: prop });
			}
		}
		return map;
	});

	const availableLotsForSelectedEntity = $derived(
		properties
			.filter((p) => !leaseEntityId || p.legal_entity_id === leaseEntityId)
			.flatMap((p) => (p.lots ?? []).map((lot) => ({ lot, property: p })))
	);

	function formatLeaseTypeLabel(lt: string): string {
		return LEASE_TYPES.find((item) => item.value === lt)?.label ?? lt;
	}

	async function handleCreateTenant(event: SubmitEvent) {
		event.preventDefault();
		if (!tenantFullName.trim()) return;

		isCreatingTenant = true;
		try {
			const created = await apiRegistry.lease.createTenant({
				full_name: tenantFullName.trim(),
				email: tenantEmail.trim(),
				phone: tenantPhone.trim(),
				address: tenantAddress.trim()
			});
			toast.success(`Locataire « ${created.full_name} » enregistré.`);
			createTenantOpen = false;
			tenantFullName = '';
			tenantEmail = '';
			tenantPhone = '';
			tenantAddress = '';
			leaseTenantId = created.id;
			reloadCounter += 1;
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Échec de la création du locataire.');
		} finally {
			isCreatingTenant = false;
		}
	}

	async function handleArchiveTenant(tenant: TenantDTO) {
		try {
			await apiRegistry.lease.archiveTenant(tenant.id);
			toast.success(`Locataire « ${tenant.full_name} » archivé (conformité RGPD).`);
			reloadCounter += 1;
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Impossible d’archiver ce locataire.');
		}
	}

	async function handleCreateLease(event: SubmitEvent) {
		event.preventDefault();
		if (!leaseEntityId || !leaseLotId || !leaseTenantId) {
			toast.error('Veuillez sélectionner une entité, un lot et un locataire.');
			return;
		}

		let rentNetCents = 0;
		let chargesCents = 0;
		let depositCents = 0;
		let vatBasisPoints = 0;

		try {
			rentNetCents = parseEuroToCents(leaseRentNetEuro);
			chargesCents = parseEuroToCents(leaseChargesEuro || '0');
			depositCents = parseEuroToCents(leaseDepositEuro || '0');
			if (leaseSubjectToVat) {
				vatBasisPoints = parsePercentToBasisPoints(leaseVatPercent);
			}
		} catch {
			toast.error('Montant de loyer, charges, dépôt ou taux de TVA invalide.');
			return;
		}

		isCreatingLease = true;
		try {
			await apiRegistry.lease.createLease({
				legal_entity_id: leaseEntityId,
				lot_id: leaseLotId,
				tenant_id: leaseTenantId,
				lease_type: leaseType,
				rent_net_cents: rentNetCents,
				charges_cents: chargesCents,
				subject_to_vat: leaseSubjectToVat,
				vat_rate_basis_points: vatBasisPoints,
				security_deposit_cents: depositCents,
				due_day_of_month: Number.parseInt(leaseDueDay, 10) || 5,
				gli_policy_number: leaseGliPolicy.trim(),
				guarantor_name: leaseGuarantor.trim(),
				start_date: leaseStartDate ? new Date(leaseStartDate).toISOString() : undefined
			});
			toast.success('Contrat de bail créé.');
			createLeaseOpen = false;
			reloadCounter += 1;
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Impossible de créer le bail.');
		} finally {
			isCreatingLease = false;
		}
	}

	async function openLeaseDocumentsSheet(lease: LeaseDTO) {
		selectedLease = lease;
		leaseDocsSheetOpen = true;
		selectedLeaseDocs = await apiRegistry.document.listLeaseDocuments(lease.id).catch(() => []);
	}

	async function handleGenerateLeaseDoc() {
		if (!selectedLease || !selectedTemplateId) return;
		isGeneratingDoc = true;
		try {
			const tenant = tenantMap.get(selectedLease.tenant_id);
			const entity = entityMap.get(selectedLease.legal_entity_id);
			const lotInfo = lotInfoMap.get(selectedLease.lot_id);

			const generated = await apiRegistry.document.generateFromTemplate({
				template_id: selectedTemplateId,
				legal_entity_id: selectedLease.legal_entity_id,
				lease_id: selectedLease.id,
				variables: {
					tenant_name: tenant?.full_name ?? 'Locataire',
					entity_name: entity?.name ?? 'Bailleur',
					property_label: lotInfo
						? `${lotInfo.property.label} — ${lotInfo.lot.label}`
						: selectedLease.lot_id,
					rent_net: formatCentsToEuro(selectedLease.rent_net_cents),
					charges: formatCentsToEuro(selectedLease.charges_cents)
				}
			});
			selectedLeaseDocs = [generated, ...selectedLeaseDocs];
			toast.success(`Document généré : ${generated.file_name}`);
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Échec de la génération.');
		} finally {
			isGeneratingDoc = false;
		}
	}

	async function handleUploadLeaseFile(event: SubmitEvent) {
		event.preventDefault();
		if (!selectedLease || !uploadFile) return;

		isUploadingDoc = true;
		try {
			const uploaded = await apiRegistry.document.uploadFile({
				legalEntityId: selectedLease.legal_entity_id,
				leaseId: selectedLease.id,
				category: uploadCategory,
				file: uploadFile
			});
			selectedLeaseDocs = [uploaded, ...selectedLeaseDocs];
			uploadFile = null;
			toast.success('Pièce justificative téléversée dans le coffre-fort R2.');
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Échec du téléversement.');
		} finally {
			isUploadingDoc = false;
		}
	}

	async function handleDocxFileInput(event: Event) {
		const input = event.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		if (!file) return;
		const buffer = await file.arrayBuffer();
		const bytes = new Uint8Array(buffer);
		let binary = '';
		for (const b of bytes) {
			binary += String.fromCharCode(b);
		}
		tplDocxBase64 = btoa(binary);
	}

	async function handleRegisterTemplate(event: SubmitEvent) {
		event.preventDefault();
		if (!tplName.trim()) return;

		isRegisteringTemplate = true;
		try {
			const created = await apiRegistry.document.registerTemplate({
				name: tplName.trim(),
				template_type: tplType,
				format: tplFormat,
				html_body: tplFormat === 'HTML' ? tplHtmlBody : undefined,
				docx_base64: tplFormat === 'DOCX' ? tplDocxBase64 : undefined
			});
			templates = [created, ...templates];
			selectedTemplateId = created.id;
			registerTemplateOpen = false;
			tplName = '';
			toast.success('Modèle documentaire enregistré.');
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Impossible d’enregistrer le modèle.');
		} finally {
			isRegisteringTemplate = false;
		}
	}
</script>

<PageHeader
	title="Baux, Locataires & Contrats"
	subtitle="Contrats de location (Loi 89, Meublé, Mobilité, Commercial 3/6/9), fiches locataires RGPD et modèles .docx / PDF"
>
	{#snippet actions()}
		<Button size="sm" variant="outline" onclick={() => (registerTemplateOpen = true)}>
			<IconFileDescription data-icon="inline-start" />
			Nouveau modèle (.docx / HTML)
		</Button>
		<Button size="sm" variant="secondary" onclick={() => (createTenantOpen = true)}>
			<IconUserPlus data-icon="inline-start" />
			Nouveau locataire
		</Button>
		<Button size="sm" onclick={() => (createLeaseOpen = true)}>
			<IconPlus data-icon="inline-start" />
			Nouveau bail
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
		<Tabs.Trigger value="leases">Baux locatifs ({leases.length})</Tabs.Trigger>
		<Tabs.Trigger value="tenants">Locataires & RGPD ({tenants.length})</Tabs.Trigger>
		<Tabs.Trigger value="templates">Modèles de contrats ({templates.length})</Tabs.Trigger>
	</Tabs.List>

	<!-- Tab 1: Leases -->
	<Tabs.Content value="leases" class="mt-4">
		{#if isLoading}
			<Skeleton class="h-48 w-full" />
		{:else if leases.length === 0}
			<Card.Root>
				<Card.Content class="py-10">
					<Empty.Root>
						<Empty.Header>
							<Empty.Media variant="icon">
								<IconFileDescription class="size-5" />
							</Empty.Media>
							<Empty.Title>Aucun contrat de bail enregistré</Empty.Title>
							<Empty.Description>
								Créez un locataire puis associez-le à un lot immobilier pour activer la génération
								des appels de loyers.
							</Empty.Description>
						</Empty.Header>
						<Empty.Content>
							<Button onclick={() => (createLeaseOpen = true)}>
								<IconPlus data-icon="inline-start" />
								Créer un bail locatif
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
								<Table.Head class="pl-4">Bien & Lot</Table.Head>
								<Table.Head>Locataire</Table.Head>
								<Table.Head>Régime du bail</Table.Head>
								<Table.Head class="text-right">Loyer HC + Charges</Table.Head>
								<Table.Head>TVA / GLI</Table.Head>
								<Table.Head class="pr-4 text-right">Documents & Loyers</Table.Head>
							</Table.Row>
						</Table.Header>
						<Table.Body>
							{#each leases as lease (lease.id)}
								{@const lotInfo = lotInfoMap.get(lease.lot_id)}
								{@const tenant = tenantMap.get(lease.tenant_id)}
								<Table.Row>
									<Table.Cell class="pl-4">
										<div class="flex flex-col">
											<span class="font-medium">
												{#if lotInfo}
													{lotInfo.property.label} — {lotInfo.lot.label}
												{:else}
													Lot {lease.lot_id.slice(0, 8)}
												{/if}
											</span>
											<span class="text-muted-foreground text-[11px]">
												{entityMap.get(lease.legal_entity_id)?.name ?? ''} · Échéance le
												{lease.due_day_of_month} du mois
											</span>
										</div>
									</Table.Cell>
									<Table.Cell class="font-medium">
										{tenant?.full_name ?? 'Locataire'}
									</Table.Cell>
									<Table.Cell>
										<Badge variant="outline">{formatLeaseTypeLabel(lease.lease_type)}</Badge>
									</Table.Cell>
									<Table.Cell class="text-right font-mono tabular-nums">
										{formatCentsToEuro(lease.rent_net_cents)}
										<span class="text-muted-foreground block text-[11px]">
											+ {formatCentsToEuro(lease.charges_cents)} prov.
										</span>
									</Table.Cell>
									<Table.Cell class="text-xs">
										{#if lease.subject_to_vat}
											<Badge variant="secondary">
												TVA {formatBasisPointsToPercent(lease.vat_rate_basis_points)}
											</Badge>
										{/if}
										{#if lease.gli_policy_number}
											<span class="text-muted-foreground block font-mono text-[11px]">
												GLI {lease.gli_policy_number}
											</span>
										{/if}
									</Table.Cell>
									<Table.Cell class="pr-4 text-right">
										<div class="flex items-center justify-end gap-1.5">
											<Button
												size="sm"
												variant="outline"
												onclick={() => openLeaseDocumentsSheet(lease)}
											>
												Pièces / Contrat →
											</Button>
											<Button
												size="sm"
												variant="secondary"
												href={`/rents?entityId=${encodeURIComponent(lease.legal_entity_id)}&leaseId=${encodeURIComponent(lease.id)}`}
											>
												Loyers →
											</Button>
										</div>
									</Table.Cell>
								</Table.Row>
							{/each}
						</Table.Body>
					</Table.Root>
				</Card.Content>
			</Card.Root>
		{/if}
	</Tabs.Content>

	<!-- Tab 2: Tenants & GDPR -->
	<Tabs.Content value="tenants" class="mt-4">
		<Card.Root>
			<Card.Content class="p-0">
				{#if tenants.length === 0}
					<p class="text-muted-foreground px-4 py-6 text-xs">Aucun locataire enregistré.</p>
				{:else}
					<Table.Root>
						<Table.Header>
							<Table.Row>
								<Table.Head class="pl-4">Nom complet</Table.Head>
								<Table.Head>Email & Téléphone</Table.Head>
								<Table.Head>Adresse</Table.Head>
								<Table.Head>Statut RGPD</Table.Head>
								<Table.Head class="pr-4 text-right">Action</Table.Head>
							</Table.Row>
						</Table.Header>
						<Table.Body>
							{#each tenants as tenant (tenant.id)}
								<Table.Row>
									<Table.Cell class="pl-4 font-medium">{tenant.full_name}</Table.Cell>
									<Table.Cell class="text-xs">
										<span>{tenant.email}</span>
										{#if tenant.phone}
											<span class="text-muted-foreground block">{tenant.phone}</span>
										{/if}
									</Table.Cell>
									<Table.Cell class="text-muted-foreground text-xs">
										{tenant.address || '—'}
									</Table.Cell>
									<Table.Cell>
										{#if tenant.archived}
											<Badge variant="outline">Archivé RGPD</Badge>
										{:else}
											<Badge variant="secondary">Actif</Badge>
										{/if}
									</Table.Cell>
									<Table.Cell class="pr-4 text-right">
										{#if !tenant.archived}
											<Button
												size="sm"
												variant="outline"
												onclick={() => handleArchiveTenant(tenant)}
											>
												<IconArchive data-icon="inline-start" />
												Archiver RGPD
											</Button>
										{/if}
									</Table.Cell>
								</Table.Row>
							{/each}
						</Table.Body>
					</Table.Root>
				{/if}
			</Card.Content>
		</Card.Root>
	</Tabs.Content>

	<!-- Tab 3: Document Templates -->
	<Tabs.Content value="templates" class="mt-4">
		<Card.Root>
			<Card.Content class="p-0">
				{#if templates.length === 0}
					<p class="text-muted-foreground px-4 py-6 text-xs">
						Aucun modèle documentaire enregistré. Cliquez sur « Nouveau modèle (.docx / HTML) ».
					</p>
				{:else}
					<Table.Root>
						<Table.Header>
							<Table.Row>
								<Table.Head class="pl-4">Nom du modèle</Table.Head>
								<Table.Head>Catégorie</Table.Head>
								<Table.Head>Format moteur</Table.Head>
								<Table.Head class="pr-4 text-right">Créé le</Table.Head>
							</Table.Row>
						</Table.Header>
						<Table.Body>
							{#each templates as tpl (tpl.id)}
								<Table.Row>
									<Table.Cell class="pl-4 font-medium">{tpl.name}</Table.Cell>
									<Table.Cell>
										<Badge variant="outline">{tpl.template_type}</Badge>
									</Table.Cell>
									<Table.Cell>
										<Badge variant="secondary">{tpl.format}</Badge>
									</Table.Cell>
									<Table.Cell class="pr-4 text-right text-xs">
										{formatIsoDateFr(tpl.created_at)}
									</Table.Cell>
								</Table.Row>
							{/each}
						</Table.Body>
					</Table.Root>
				{/if}
			</Card.Content>
		</Card.Root>
	</Tabs.Content>
</Tabs.Root>

<!-- Create Lease Sheet -->
<Sheet.Root bind:open={createLeaseOpen}>
	<Sheet.Content side="right" class="w-full overflow-y-auto sm:max-w-lg">
		<Sheet.Header class="border-b pb-4">
			<Sheet.Title>Nouveau contrat de bail</Sheet.Title>
			<Sheet.Description>
				Paramétrez le loyer hors charges, les provisions sur charges, l'option TVA et la police GLI.
			</Sheet.Description>
		</Sheet.Header>

		<form onsubmit={handleCreateLease} class="flex flex-col gap-4 p-4">
			<Field.FieldGroup>
				<Field.Field>
					<Field.Label>Entité bailleresse</Field.Label>
					<Select.Root
						type="single"
						value={leaseEntityId}
						onValueChange={(v) => {
							if (v) leaseEntityId = v;
						}}
					>
						<Select.Trigger class="w-full">
							{entityMap.get(leaseEntityId)?.name ?? 'Sélectionner une entité'}
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
					<Field.Label>Lot immobilier loué</Field.Label>
					<Select.Root
						type="single"
						value={leaseLotId}
						onValueChange={(v) => {
							if (v) leaseLotId = v;
						}}
					>
						<Select.Trigger class="w-full">
							{#if lotInfoMap.get(leaseLotId)}
								{lotInfoMap.get(leaseLotId)?.property.label} —
								{lotInfoMap.get(leaseLotId)?.lot.label}
							{:else}
								Sélectionner un lot
							{/if}
						</Select.Trigger>
						<Select.Content>
							<Select.Group>
								{#each availableLotsForSelectedEntity as item (item.lot.id)}
									<Select.Item
										value={item.lot.id}
										label={`${item.property.label} — ${item.lot.label}`}
									>
										{item.property.label} — {item.lot.label}
									</Select.Item>
								{/each}
							</Select.Group>
						</Select.Content>
					</Select.Root>
				</Field.Field>

				<Field.Field>
					<Field.Label>Locataire titulaire</Field.Label>
					<Select.Root
						type="single"
						value={leaseTenantId}
						onValueChange={(v) => {
							if (v) leaseTenantId = v;
						}}
					>
						<Select.Trigger class="w-full">
							{tenantMap.get(leaseTenantId)?.full_name ?? 'Sélectionner un locataire'}
						</Select.Trigger>
						<Select.Content>
							<Select.Group>
								{#each tenants.filter((t) => !t.archived) as tenant (tenant.id)}
									<Select.Item value={tenant.id} label={tenant.full_name}>
										{tenant.full_name}
									</Select.Item>
								{/each}
							</Select.Group>
						</Select.Content>
					</Select.Root>
				</Field.Field>

				<Field.Field>
					<Field.Label>Régime juridique du bail</Field.Label>
					<Select.Root
						type="single"
						value={leaseType}
						onValueChange={(v) => {
							if (v) leaseType = v;
						}}
					>
						<Select.Trigger class="w-full">
							{formatLeaseTypeLabel(leaseType)}
						</Select.Trigger>
						<Select.Content>
							<Select.Group>
								{#each LEASE_TYPES as lt (lt.value)}
									<Select.Item value={lt.value} label={lt.label}>{lt.label}</Select.Item>
								{/each}
							</Select.Group>
						</Select.Content>
					</Select.Root>
				</Field.Field>

				<div class="grid gap-3 sm:grid-cols-3">
					<Field.Field>
						<Field.Label for="l-rent">Loyer HC (€)</Field.Label>
						<Input
							id="l-rent"
							type="text"
							inputmode="decimal"
							bind:value={leaseRentNetEuro}
							required
						/>
					</Field.Field>

					<Field.Field>
						<Field.Label for="l-charges">Provisions charges (€)</Field.Label>
						<Input
							id="l-charges"
							type="text"
							inputmode="decimal"
							bind:value={leaseChargesEuro}
						/>
					</Field.Field>

					<Field.Field>
						<Field.Label for="l-dep">Dépôt garantie (€)</Field.Label>
						<Input
							id="l-dep"
							type="text"
							inputmode="decimal"
							bind:value={leaseDepositEuro}
						/>
					</Field.Field>
				</div>

				<div class="flex items-center justify-between gap-3 rounded-md border p-3">
					<div class="flex items-center gap-2">
						<Checkbox id="l-vat" bind:checked={leaseSubjectToVat} />
						<label for="l-vat" class="text-xs font-medium">Bail soumis à la TVA (option 20%)</label>
					</div>
					{#if leaseSubjectToVat}
						<Input
							class="h-8 w-24 text-right"
							type="text"
							inputmode="decimal"
							bind:value={leaseVatPercent}
							aria-label="Taux de TVA (%)"
						/>
					{/if}
				</div>

				<div class="grid gap-3 sm:grid-cols-2">
					<Field.Field>
						<Field.Label for="l-start">Date de prise d'effet</Field.Label>
						<Input id="l-start" type="date" bind:value={leaseStartDate} required />
					</Field.Field>

					<Field.Field>
						<Field.Label for="l-day">Jour d'échéance mensuel (1-28)</Field.Label>
						<Input id="l-day" type="number" min="1" max="28" bind:value={leaseDueDay} required />
					</Field.Field>
				</div>

				<div class="grid gap-3 sm:grid-cols-2">
					<Field.Field>
						<Field.Label for="l-gli">N° police assurance GLI</Field.Label>
						<Input id="l-gli" placeholder="GLI-AXA-2026-99" bind:value={leaseGliPolicy} />
					</Field.Field>

					<Field.Field>
						<Field.Label for="l-guar">Garant physique / Visale</Field.Label>
						<Input id="l-guar" placeholder="Caution solidaire" bind:value={leaseGuarantor} />
					</Field.Field>
				</div>
			</Field.FieldGroup>

			<Button type="submit" disabled={isCreatingLease}>
				{#if isCreatingLease}
					<Spinner data-icon="inline-start" />
				{/if}
				Activer le contrat de bail
			</Button>
		</form>
	</Sheet.Content>
</Sheet.Root>

<!-- Create Tenant Sheet -->
<Sheet.Root bind:open={createTenantOpen}>
	<Sheet.Content side="right" class="w-full overflow-y-auto sm:max-w-md">
		<Sheet.Header class="border-b pb-4">
			<Sheet.Title>Nouveau locataire</Sheet.Title>
			<Sheet.Description>
				Créez une fiche locataire pour l'associer à un bail d'habitation ou commercial.
			</Sheet.Description>
		</Sheet.Header>

		<form onsubmit={handleCreateTenant} class="flex flex-col gap-4 p-4">
			<Field.FieldGroup>
				<Field.Field>
					<Field.Label for="t-name">Nom complet / Raison sociale</Field.Label>
					<Input
						id="t-name"
						placeholder="Claire Martin"
						bind:value={tenantFullName}
						required
					/>
				</Field.Field>

				<Field.Field>
					<Field.Label for="t-email">Adresse e-mail</Field.Label>
					<Input
						id="t-email"
						type="email"
						placeholder="claire.martin@exemple.fr"
						bind:value={tenantEmail}
						required
					/>
				</Field.Field>

				<Field.Field>
					<Field.Label for="t-phone">Téléphone</Field.Label>
					<Input
						id="t-phone"
						type="tel"
						placeholder="06 12 34 56 78"
						bind:value={tenantPhone}
					/>
				</Field.Field>

				<Field.Field>
					<Field.Label for="t-addr">Adresse postale</Field.Label>
					<Input
						id="t-addr"
						placeholder="12 rue de Rivoli, 75004 Paris"
						bind:value={tenantAddress}
					/>
				</Field.Field>
			</Field.FieldGroup>

			<Button type="submit" disabled={isCreatingTenant}>
				{#if isCreatingTenant}
					<Spinner data-icon="inline-start" />
				{/if}
				Enregistrer le locataire
			</Button>
		</form>
	</Sheet.Content>
</Sheet.Root>

<!-- Lease Documents & Upload Sheet -->
<Sheet.Root bind:open={leaseDocsSheetOpen}>
	<Sheet.Content side="right" class="w-full overflow-y-auto sm:max-w-lg">
		<Sheet.Header class="border-b pb-4">
			<Sheet.Title>
				Contrat & Coffre-fort documentaire R2
			</Sheet.Title>
			<Sheet.Description>
				Générez un document à partir d'un modèle ou téléversez une pièce justificative (bail signé,
				attestation d'assurance, RIB).
			</Sheet.Description>
		</Sheet.Header>

		{#if selectedLease}
			<div class="flex flex-col gap-6 p-4">
				<!-- Generate from template -->
				<div class="flex flex-col gap-3">
					<h3 class="text-xs font-semibold uppercase">1. Générer depuis un modèle</h3>
					{#if templates.length > 0}
						<div class="flex gap-2">
							<div class="flex-1">
								<Select.Root
									type="single"
									value={selectedTemplateId}
									onValueChange={(v) => {
										if (v) selectedTemplateId = v;
									}}
								>
									<Select.Trigger class="w-full">
										{templates.find((t) => t.id === selectedTemplateId)?.name ??
											'Sélectionner un modèle'}
									</Select.Trigger>
									<Select.Content>
										<Select.Group>
											{#each templates as tpl (tpl.id)}
												<Select.Item value={tpl.id} label={tpl.name}>
													{tpl.name} ({tpl.format})
												</Select.Item>
											{/each}
										</Select.Group>
									</Select.Content>
								</Select.Root>
							</div>
							<Button
								variant="secondary"
								onclick={handleGenerateLeaseDoc}
								disabled={isGeneratingDoc || !selectedTemplateId}
							>
								{#if isGeneratingDoc}
									<Spinner data-icon="inline-start" />
								{/if}
								Générer
							</Button>
						</div>
					{:else}
						<p class="text-muted-foreground text-xs">Aucun modèle enregistré.</p>
					{/if}
				</div>

				<Separator />

				<!-- Upload file to R2 -->
				<form onsubmit={handleUploadLeaseFile} class="flex flex-col gap-3">
					<h3 class="text-xs font-semibold uppercase">2. Téléverser une pièce (Cloudflare R2)</h3>
					<Field.FieldGroup>
						<Field.Field>
							<Field.Label for="up-cat">Catégorie de pièce</Field.Label>
							<Input
								id="up-cat"
								placeholder="BAIL_SIGNE, ATTESTATION_ASSURANCE, CAUTION..."
								bind:value={uploadCategory}
								required
							/>
						</Field.Field>

						<Field.Field>
							<Field.Label for="up-file">Fichier PDF / DOCX / Image</Field.Label>
							<Input
								id="up-file"
								type="file"
								onchange={(e) => {
									uploadFile = (e.currentTarget as HTMLInputElement).files?.[0] ?? null;
								}}
								required
							/>
						</Field.Field>
					</Field.FieldGroup>

					<Button type="submit" variant="outline" disabled={isUploadingDoc || !uploadFile}>
						{#if isUploadingDoc}
							<Spinner data-icon="inline-start" />
						{:else}
							<IconFileUpload data-icon="inline-start" />
						{/if}
						Téléverser dans le dossier du bail
					</Button>
				</form>

				<Separator />

				<!-- Existing documents -->
				<div class="flex flex-col gap-2">
					<h3 class="text-xs font-semibold uppercase">
						3. Pièces archivées ({selectedLeaseDocs.length})
					</h3>
					{#if selectedLeaseDocs.length === 0}
						<p class="text-muted-foreground text-xs">Aucune pièce archivée sur ce bail.</p>
					{:else}
						{#each selectedLeaseDocs as doc (doc.id)}
							<div class="flex items-center justify-between rounded-md border p-2.5 text-xs">
								<div class="flex min-w-0 flex-col">
									<span class="truncate font-medium">{doc.file_name}</span>
									<span class="text-muted-foreground text-[11px]">
										{doc.category} · {formatIsoDateFr(doc.created_at)}
									</span>
								</div>
								{#if doc.download_url}
									<Button
										size="sm"
										variant="outline"
										href={doc.download_url}
										target="_blank"
										rel="noreferrer"
									>
										<IconDownload data-icon="inline-start" />
										Télécharger
									</Button>
								{/if}
							</div>
						{/each}
					{/if}
				</div>
			</div>
		{/if}
	</Sheet.Content>
</Sheet.Root>

<!-- Register Document Template Dialog -->
<Dialog.Root bind:open={registerTemplateOpen}>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>Enregistrer un modèle de document</Dialog.Title>
			<Dialog.Description>
				Supporte les modèles Word <code>.docx</code> avec balises <code>&#123;&#123;variable&#125;&#125;</code>
				et les gabarits HTML convertis en PDF.
			</Dialog.Description>
		</Dialog.Header>

		<form onsubmit={handleRegisterTemplate} class="flex flex-col gap-4">
			<Field.FieldGroup>
				<Field.Field>
					<Field.Label for="reg-tpl-name">Nom du modèle</Field.Label>
					<Input
						id="reg-tpl-name"
						placeholder="Bail Habitation Loi 89 Standard"
						bind:value={tplName}
						required
					/>
				</Field.Field>

				<div class="grid gap-3 sm:grid-cols-2">
					<Field.Field>
						<Field.Label>Catégorie</Field.Label>
						<Select.Root
							type="single"
							value={tplType}
							onValueChange={(v) => {
								if (v) tplType = v;
							}}
						>
							<Select.Trigger class="w-full">
								{TEMPLATE_TYPES.find((t) => t.value === tplType)?.label ?? tplType}
							</Select.Trigger>
							<Select.Content>
								<Select.Group>
									{#each TEMPLATE_TYPES as item (item.value)}
										<Select.Item value={item.value} label={item.label}>{item.label}</Select.Item>
									{/each}
								</Select.Group>
							</Select.Content>
						</Select.Root>
					</Field.Field>

					<Field.Field>
						<Field.Label>Format</Field.Label>
						<Select.Root
							type="single"
							value={tplFormat}
							onValueChange={(v) => {
								if (v) tplFormat = v;
							}}
						>
							<Select.Trigger class="w-full">{tplFormat}</Select.Trigger>
							<Select.Content>
								<Select.Group>
									<Select.Item value="HTML" label="HTML → PDF">HTML → PDF</Select.Item>
									<Select.Item value="DOCX" label="Word .docx">Word .docx</Select.Item>
								</Select.Group>
							</Select.Content>
						</Select.Root>
					</Field.Field>
				</div>

				{#if tplFormat === 'HTML'}
					<Field.Field>
						<Field.Label for="reg-tpl-html">Gabarit HTML</Field.Label>
						<textarea
							id="reg-tpl-html"
							rows="4"
							class="border-input bg-background w-full rounded-md border p-2 font-mono text-xs"
							bind:value={tplHtmlBody}
							required
						></textarea>
					</Field.Field>
				{:else}
					<Field.Field>
						<Field.Label for="reg-tpl-docx">Fichier .docx source</Field.Label>
						<Input
							id="reg-tpl-docx"
							type="file"
							accept=".docx"
							onchange={handleDocxFileInput}
							required
						/>
					</Field.Field>
				{/if}
			</Field.FieldGroup>

			<Dialog.Footer>
				<Button variant="outline" type="button" onclick={() => (registerTemplateOpen = false)}>
					Annuler
				</Button>
				<Button type="submit" disabled={isRegisteringTemplate}>
					{#if isRegisteringTemplate}
						<Spinner data-icon="inline-start" />
					{/if}
					Enregistrer le modèle
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
