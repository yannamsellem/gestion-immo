<script lang="ts">
	import { enhance } from '$app/forms';
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
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import * as Table from '#lib/components/ui/table/index.js';
	import * as Tabs from '#lib/components/ui/tabs/index.js';
	import { formatIsoDateFr } from '#lib/core/formatters/date';
	import { formatBasisPointsToPercent, formatCentsToEuro } from '#lib/core/formatters/money';
	import { workspaceStore } from '#lib/core/workspace/workspace.svelte';
	import type { TemplateFormat, TemplateType } from '#lib/modules/document/types';
	import type { LeaseDTO, LeaseType } from '#lib/modules/lease/types';
	import type { LotDTO, PropertyDTO } from '#lib/modules/portfolio/types';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	const LEASE_TYPES: Array<{ value: LeaseType; label: string }> = [
		{ value: 'RESIDENTIAL_UNFURNISHED', label: 'Habitation vide (Loi 89 — 3 ans)' },
		{ value: 'RESIDENTIAL_FURNISHED', label: 'Habitation meublée (1 an / 9 mois)' },
		{ value: 'COMMERCIAL_3_6_9', label: 'Bail commercial 3 / 6 / 9' },
		{ value: 'PROFESSIONAL', label: 'Bail professionnel' },
		{ value: 'SHORT_TERM', label: 'Bail mobilité / courte durée' },
		{ value: 'PARKING', label: 'Bail parking / garage' }
	];

	const TEMPLATE_TYPES: Array<{ value: TemplateType; label: string }> = [
		{ value: 'LEASE', label: 'Contrat de bail' },
		{ value: 'RENT_RECEIPT', label: 'Quittance / Avis d’échéance' },
		{ value: 'FORMAL_NOTICE', label: 'Lettre de relance / Mise en demeure' },
		{ value: 'GLI_DECLARATION', label: 'Déclaration sinistre GLI' },
		{ value: 'CUSTOM', label: 'Modèle personnalisé' }
	];

	let activeTab = $state<'leases' | 'tenants' | 'templates'>('leases');

	// Create Lease Sheet
	let createLeaseOpen = $state(false);
	let leaseEntityId = $state('');
	let leaseLotId = $state('');
	let leaseTenantId = $state('');
	let leaseType = $state<LeaseType>('RESIDENTIAL_UNFURNISHED');
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
	let selectedTemplateId = $state('');
	let uploadCategory = $state('BAIL_SIGNE');
	let hasUploadFile = $state(false);
	let isUploadingDoc = $state(false);
	let isGeneratingDoc = $state(false);

	// Register Template Dialog
	let registerTemplateOpen = $state(false);
	let tplName = $state('');
	let tplType = $state<TemplateType>('LEASE');
	let tplFormat = $state<TemplateFormat>('HTML');
	let tplHtmlBody = $state(
		'<h1>{{entity_name}}</h1><p>Locataire : {{tenant_name}}</p><p>Lot : {{property_label}}</p>'
	);
	let tplDocxBase64 = $state('');
	let isRegisteringTemplate = $state(false);

	let autoHandledUrl = $state(false);

	$effect(() => {
		if (data.entities.length > 0 && !leaseEntityId) {
			leaseEntityId =
				workspaceStore.selectedEntityId !== 'ALL'
					? workspaceStore.selectedEntityId
					: data.entities[0].id;
		}
		const allLots = data.properties.flatMap((p) => p.lots ?? []);
		if (allLots.length > 0 && !leaseLotId) {
			leaseLotId = allLots[0].id;
		}
		const activeTenants = data.tenants.filter((t) => !t.archived);
		if (activeTenants.length > 0 && !leaseTenantId) {
			leaseTenantId = activeTenants[0].id;
		}
		if (data.templates.length > 0 && !selectedTemplateId) {
			selectedTemplateId = data.templates[0].id;
		}

		if (!autoHandledUrl) {
			const action = page.url.searchParams.get('action');
			const urlLotId = page.url.searchParams.get('lotId');
			if (urlLotId) {
				leaseLotId = urlLotId;
				const ownerProp = data.properties.find((p) =>
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
	});

	$effect(() => {
		if (form?.successMessage) {
			toast.success(form.successMessage);
		}
		if (form?.actionError) {
			toast.error(form.actionError);
		}
		if (form?.createdTenant) {
			createTenantOpen = false;
			tenantFullName = '';
			tenantEmail = '';
			tenantPhone = '';
			tenantAddress = '';
			leaseTenantId = form.createdTenant.id;
		}
		if (form?.leaseCreated) {
			createLeaseOpen = false;
		}
		if (form?.registeredTemplate) {
			selectedTemplateId = form.registeredTemplate.id;
			registerTemplateOpen = false;
			tplName = '';
		}
	});

	const entityMap = $derived(new Map(data.entities.map((e) => [e.id, e])));
	const tenantMap = $derived(new Map(data.tenants.map((t) => [t.id, t])));
	const lotInfoMap = $derived(
		new Map<string, { lot: LotDTO; property: PropertyDTO }>(
			data.properties.flatMap((prop) =>
				(prop.lots ?? []).map((lot) => [lot.id, { lot, property: prop }] as const)
			)
		)
	);

	const availableLotsForSelectedEntity = $derived(
		data.properties
			.filter((p) => !leaseEntityId || p.legal_entity_id === leaseEntityId)
			.flatMap((p) => (p.lots ?? []).map((lot) => ({ lot, property: p })))
	);

	const selectedLeaseDocs = $derived(
		selectedLease ? (data.documentsByLeaseId[selectedLease.id] ?? []) : []
	);

	function formatLeaseTypeLabel(lt: string): string {
		return LEASE_TYPES.find((item) => item.value === lt)?.label ?? lt;
	}

	function openLeaseDocumentsSheet(lease: LeaseDTO) {
		selectedLease = lease;
		leaseDocsSheetOpen = true;
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

{#if data.errorMessage}
	<Alert.Root variant="destructive">
		<IconAlertTriangle class="size-4" />
		<Alert.Title>Erreur</Alert.Title>
		<Alert.Description>{data.errorMessage}</Alert.Description>
	</Alert.Root>
{/if}

<Tabs.Root bind:value={activeTab}>
	<Tabs.List>
		<Tabs.Trigger value="leases">Baux locatifs ({data.leases.length})</Tabs.Trigger>
		<Tabs.Trigger value="tenants">Locataires & RGPD ({data.tenants.length})</Tabs.Trigger>
		<Tabs.Trigger value="templates">Modèles de contrats ({data.templates.length})</Tabs.Trigger>
	</Tabs.List>

	<!-- Tab 1: Leases -->
	<Tabs.Content value="leases" class="mt-4">
		{#if data.leases.length === 0}
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
							{#each data.leases as lease (lease.id)}
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
											<span class="text-[11px] text-muted-foreground">
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
										<span class="block text-[11px] text-muted-foreground">
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
											<span class="block font-mono text-[11px] text-muted-foreground">
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
				{#if data.tenants.length === 0}
					<p class="px-4 py-6 text-xs text-muted-foreground">Aucun locataire enregistré.</p>
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
							{#each data.tenants as tenant (tenant.id)}
								<Table.Row>
									<Table.Cell class="pl-4 font-medium">{tenant.full_name}</Table.Cell>
									<Table.Cell class="text-xs">
										<span>{tenant.email}</span>
										{#if tenant.phone}
											<span class="block text-muted-foreground">{tenant.phone}</span>
										{/if}
									</Table.Cell>
									<Table.Cell class="text-xs text-muted-foreground">
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
											<form method="POST" action="?/archiveTenant" use:enhance class="inline">
												<input type="hidden" name="tenant_id" value={tenant.id} />
												<input type="hidden" name="tenant_name" value={tenant.full_name} />
												<Button type="submit" size="sm" variant="outline">
													<IconArchive data-icon="inline-start" />
													Archiver RGPD
												</Button>
											</form>
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
				{#if data.templates.length === 0}
					<p class="px-4 py-6 text-xs text-muted-foreground">
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
							{#each data.templates as tpl (tpl.id)}
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

		<form
			method="POST"
			action="?/createLease"
			use:enhance={() => {
				isCreatingLease = true;
				return async ({ update }) => {
					await update();
					isCreatingLease = false;
				};
			}}
			class="flex flex-col gap-4 p-4"
		>
			<input type="hidden" name="legal_entity_id" value={leaseEntityId} />
			<input type="hidden" name="lot_id" value={leaseLotId} />
			<input type="hidden" name="tenant_id" value={leaseTenantId} />
			<input type="hidden" name="lease_type" value={leaseType} />
			<input type="hidden" name="subject_to_vat" value={leaseSubjectToVat ? 'true' : 'false'} />

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
								{#each data.entities as entity (entity.id)}
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
								{#each data.tenants.filter((t) => !t.archived) as tenant (tenant.id)}
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
							name="rent_net_euro"
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
							name="charges_euro"
							type="text"
							inputmode="decimal"
							bind:value={leaseChargesEuro}
						/>
					</Field.Field>

					<Field.Field>
						<Field.Label for="l-dep">Dépôt garantie (€)</Field.Label>
						<Input
							id="l-dep"
							name="deposit_euro"
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
							name="vat_percent"
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
						<Input
							id="l-start"
							name="start_date"
							type="date"
							bind:value={leaseStartDate}
							required
						/>
					</Field.Field>

					<Field.Field>
						<Field.Label for="l-day">Jour d'échéance mensuel (1-28)</Field.Label>
						<Input
							id="l-day"
							name="due_day"
							type="number"
							min="1"
							max="28"
							bind:value={leaseDueDay}
							required
						/>
					</Field.Field>
				</div>

				<div class="grid gap-3 sm:grid-cols-2">
					<Field.Field>
						<Field.Label for="l-gli">N° police assurance GLI</Field.Label>
						<Input
							id="l-gli"
							name="gli_policy_number"
							placeholder="GLI-AXA-2026-99"
							bind:value={leaseGliPolicy}
						/>
					</Field.Field>

					<Field.Field>
						<Field.Label for="l-guar">Garant physique / Visale</Field.Label>
						<Input
							id="l-guar"
							name="guarantor_name"
							placeholder="Caution solidaire"
							bind:value={leaseGuarantor}
						/>
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

		<form
			method="POST"
			action="?/createTenant"
			use:enhance={() => {
				isCreatingTenant = true;
				return async ({ update }) => {
					await update();
					isCreatingTenant = false;
				};
			}}
			class="flex flex-col gap-4 p-4"
		>
			<Field.FieldGroup>
				<Field.Field>
					<Field.Label for="t-name">Nom complet / Raison sociale</Field.Label>
					<Input
						id="t-name"
						name="full_name"
						placeholder="Claire Martin"
						bind:value={tenantFullName}
						required
					/>
				</Field.Field>

				<Field.Field>
					<Field.Label for="t-email">Adresse e-mail</Field.Label>
					<Input
						id="t-email"
						name="email"
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
						name="phone"
						type="tel"
						placeholder="06 12 34 56 78"
						bind:value={tenantPhone}
					/>
				</Field.Field>

				<Field.Field>
					<Field.Label for="t-addr">Adresse postale</Field.Label>
					<Input
						id="t-addr"
						name="address"
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
			<Sheet.Title>Contrat & Coffre-fort documentaire R2</Sheet.Title>
			<Sheet.Description>
				Générez un document à partir d'un modèle ou téléversez une pièce justificative (bail signé,
				attestation d'assurance, RIB).
			</Sheet.Description>
		</Sheet.Header>

		{#if selectedLease}
			{@const tenant = tenantMap.get(selectedLease.tenant_id)}
			{@const entity = entityMap.get(selectedLease.legal_entity_id)}
			{@const lotInfo = lotInfoMap.get(selectedLease.lot_id)}
			<div class="flex flex-col gap-6 p-4">
				<!-- Generate from template -->
				<div class="flex flex-col gap-3">
					<h3 class="text-xs font-semibold uppercase">1. Générer depuis un modèle</h3>
					{#if data.templates.length > 0}
						<form
							method="POST"
							action="?/generateDoc"
							use:enhance={() => {
								isGeneratingDoc = true;
								return async ({ update }) => {
									await update();
									isGeneratingDoc = false;
								};
							}}
							class="flex gap-2"
						>
							<input type="hidden" name="lease_id" value={selectedLease.id} />
							<input type="hidden" name="legal_entity_id" value={selectedLease.legal_entity_id} />
							<input type="hidden" name="template_id" value={selectedTemplateId} />
							<input type="hidden" name="tenant_name" value={tenant?.full_name ?? 'Locataire'} />
							<input type="hidden" name="entity_name" value={entity?.name ?? 'Bailleur'} />
							<input
								type="hidden"
								name="property_label"
								value={lotInfo
									? `${lotInfo.property.label} — ${lotInfo.lot.label}`
									: selectedLease.lot_id}
							/>
							<input type="hidden" name="rent_net_cents" value={selectedLease.rent_net_cents} />
							<input type="hidden" name="charges_cents" value={selectedLease.charges_cents} />

							<div class="flex-1">
								<Select.Root
									type="single"
									value={selectedTemplateId}
									onValueChange={(v) => {
										if (v) selectedTemplateId = v;
									}}
								>
									<Select.Trigger class="w-full">
										{data.templates.find((t) => t.id === selectedTemplateId)?.name ??
											'Sélectionner un modèle'}
									</Select.Trigger>
									<Select.Content>
										<Select.Group>
											{#each data.templates as tpl (tpl.id)}
												<Select.Item value={tpl.id} label={tpl.name}>
													{tpl.name} ({tpl.format})
												</Select.Item>
											{/each}
										</Select.Group>
									</Select.Content>
								</Select.Root>
							</div>
							<Button
								type="submit"
								variant="secondary"
								disabled={isGeneratingDoc || !selectedTemplateId}
							>
								{#if isGeneratingDoc}
									<Spinner data-icon="inline-start" />
								{/if}
								Générer
							</Button>
						</form>
					{:else}
						<p class="text-xs text-muted-foreground">Aucun modèle enregistré.</p>
					{/if}
				</div>

				<Separator />

				<!-- Upload file to R2 -->
				<form
					method="POST"
					action="?/uploadFile"
					enctype="multipart/form-data"
					use:enhance={() => {
						isUploadingDoc = true;
						return async ({ update }) => {
							await update();
							isUploadingDoc = false;
							hasUploadFile = false;
						};
					}}
					class="flex flex-col gap-3"
				>
					<input type="hidden" name="lease_id" value={selectedLease.id} />
					<input type="hidden" name="legal_entity_id" value={selectedLease.legal_entity_id} />

					<h3 class="text-xs font-semibold uppercase">2. Téléverser une pièce (Cloudflare R2)</h3>
					<Field.FieldGroup>
						<Field.Field>
							<Field.Label for="up-cat">Catégorie de pièce</Field.Label>
							<Input
								id="up-cat"
								name="category"
								placeholder="BAIL_SIGNE, ATTESTATION_ASSURANCE, CAUTION..."
								bind:value={uploadCategory}
								required
							/>
						</Field.Field>

						<Field.Field>
							<Field.Label for="up-file">Fichier PDF / DOCX / Image</Field.Label>
							<Input
								id="up-file"
								name="file"
								type="file"
								onchange={(e) => {
									hasUploadFile = Boolean((e.currentTarget as HTMLInputElement).files?.length);
								}}
								required
							/>
						</Field.Field>
					</Field.FieldGroup>

					<Button type="submit" variant="outline" disabled={isUploadingDoc || !hasUploadFile}>
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
						<p class="text-xs text-muted-foreground">Aucune pièce archivée sur ce bail.</p>
					{:else}
						{#each selectedLeaseDocs as doc (doc.id)}
							<div class="flex items-center justify-between rounded-md border p-2.5 text-xs">
								<div class="flex min-w-0 flex-col">
									<span class="truncate font-medium">{doc.file_name}</span>
									<span class="text-[11px] text-muted-foreground">
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
				Supporte les modèles Word <code>.docx</code> avec balises
				<code>&#123;&#123;variable&#125;&#125;</code>
				et les gabarits HTML convertis en PDF.
			</Dialog.Description>
		</Dialog.Header>

		<form
			method="POST"
			action="?/registerTemplate"
			use:enhance={() => {
				isRegisteringTemplate = true;
				return async ({ update }) => {
					await update();
					isRegisteringTemplate = false;
				};
			}}
			class="flex flex-col gap-4"
		>
			<input type="hidden" name="template_type" value={tplType} />
			<input type="hidden" name="format" value={tplFormat} />
			<input type="hidden" name="docx_base64" value={tplDocxBase64} />

			<Field.FieldGroup>
				<Field.Field>
					<Field.Label for="reg-tpl-name">Nom du modèle</Field.Label>
					<Input
						id="reg-tpl-name"
						name="name"
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
							name="html_body"
							rows="4"
							class="w-full rounded-md border border-input bg-background p-2 font-mono text-xs"
							bind:value={tplHtmlBody}
							required></textarea>
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
