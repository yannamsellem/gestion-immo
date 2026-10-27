<script lang="ts">
	import { enhance } from '$app/forms';
	import IconCash from '@tabler/icons-svelte/icons/cash';
	import IconDownload from '@tabler/icons-svelte/icons/download';
	import IconFileText from '@tabler/icons-svelte/icons/file-text';
	import IconFileZip from '@tabler/icons-svelte/icons/file-zip';
	import IconShieldExclamation from '@tabler/icons-svelte/icons/shield-exclamation';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import * as Select from '#lib/components/ui/select/index.js';
	import { Separator } from '#lib/components/ui/separator/index.js';
	import * as Sheet from '#lib/components/ui/sheet/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import { formatIsoDateFr, formatPeriodMonthFr } from '#lib/core/formatters/date';
	import { formatCentsToEuro } from '#lib/core/formatters/money';
	import type {
		DocumentTemplateDTO,
		GLIDossierArchiveDTO,
		StoredFileDTO
	} from '#lib/modules/document/types';
	import type { LeaseDTO, RentCallDTO } from '#lib/modules/lease/types';
	import type { NegativePositionDTO, PaymentMethod } from '#lib/modules/payment/types';

	let {
		open = $bindable(false),
		lease,
		rentCall,
		negativePosition,
		tenantName,
		entityName,
		propertyAndLotLabel,
		periodMonth,
		templates = [],
		leaseDocuments = [],
		assembledGliArchive = null,
		initialAction = null
	}: {
		open: boolean;
		lease: LeaseDTO | null;
		rentCall: RentCallDTO | null;
		negativePosition: NegativePositionDTO | null;
		tenantName: string;
		entityName: string;
		propertyAndLotLabel: string;
		periodMonth: string;
		templates?: DocumentTemplateDTO[];
		leaseDocuments?: StoredFileDTO[];
		assembledGliArchive?: GLIDossierArchiveDTO | null;
		initialAction?: 'pay' | 'remind' | 'generate-calls' | null;
	} = $props();

	const PAYMENT_METHODS: Array<{ value: PaymentMethod; label: string }> = [
		{ value: 'BANK_TRANSFER', label: 'Virement bancaire' },
		{ value: 'DIRECT_DEBIT', label: 'Prélèvement automatique' },
		{ value: 'CHECK', label: 'Chèque' },
		{ value: 'CASH', label: 'Espèces (Livre de caisse PCG 530)' }
	];

	let amountEuroInput = $state('');
	let paymentMethod = $state<PaymentMethod>('BANK_TRANSFER');
	let paymentDate = $state(new Date().toISOString().slice(0, 10));
	let paymentLabel = $state('');
	let paymentReference = $state('');
	let isSubmittingPayment = $state(false);

	let selectedTemplateId = $state('');
	let isGeneratingDoc = $state(false);
	let isAssemblingGli = $state(false);

	const selectedMethodLabel = $derived(
		PAYMENT_METHODS.find((item) => item.value === paymentMethod)?.label ?? 'Virement bancaire'
	);

	const selectedTemplateLabel = $derived(
		templates.find((tpl) => tpl.id === selectedTemplateId)?.name ?? 'Choisir un modèle de document'
	);

	const gliTemplateId = $derived(
		templates.find((tpl) => tpl.template_type === 'GLI_NOTICE' || tpl.template_type === 'REMINDER')
			?.id ?? selectedTemplateId
	);

	const formattedAmountDue = $derived(
		lease
			? formatCentsToEuro(
					rentCall?.remaining_due_cents ??
						negativePosition?.total_unpaid_cents ??
						lease.rent_net_cents + lease.charges_cents
				)
			: '0,00 €'
	);

	function centsToEditableEuroString(cents: number): string {
		return (cents / 100).toFixed(2).replace('.', ',');
	}

	$effect(() => {
		if (!open || !lease) {
			return;
		}

		const defaultDueCents =
			rentCall && rentCall.remaining_due_cents > 0
				? rentCall.remaining_due_cents
				: negativePosition && negativePosition.total_unpaid_cents > 0
					? negativePosition.total_unpaid_cents
					: rentCall
						? rentCall.total_due_cents
						: lease.rent_net_cents + lease.charges_cents;

		amountEuroInput = centsToEditableEuroString(defaultDueCents);
		paymentLabel = tenantName
			? `Loyer ${formatPeriodMonthFr(periodMonth)} — ${tenantName}`
			: `Loyer ${formatPeriodMonthFr(periodMonth)}`;
		paymentReference = '';

		if (templates.length > 0 && !selectedTemplateId) {
			const preferredType = initialAction === 'remind' ? 'REMINDER' : 'RECEIPT';
			const preferred =
				templates.find((tpl) => tpl.template_type === preferredType) ?? templates[0];
			selectedTemplateId = preferred.id;
		}
	});
</script>

<Sheet.Root bind:open>
	<Sheet.Content side="right" class="w-full overflow-y-auto sm:max-w-lg">
		<Sheet.Header class="border-b pb-4">
			<div class="flex flex-wrap items-center gap-2">
				<Badge variant="outline">{entityName}</Badge>
				<Badge variant="secondary">{formatPeriodMonthFr(periodMonth)}</Badge>
			</div>
			<Sheet.Title class="text-base font-semibold">
				{tenantName || 'Bail sélectionné'} — {propertyAndLotLabel}
			</Sheet.Title>
			<Sheet.Description>
				{#if rentCall}
					Échéance du mois : {formatCentsToEuro(rentCall.total_due_cents)} · Reste dû :
					<strong class="font-semibold text-foreground">
						{formatCentsToEuro(rentCall.remaining_due_cents)}
					</strong>
				{:else if lease}
					Loyer mensuel HC : {formatCentsToEuro(lease.rent_net_cents)} + Charges :
					{formatCentsToEuro(lease.charges_cents)}
				{/if}
			</Sheet.Description>
		</Sheet.Header>

		{#if lease}
			<div class="flex flex-col gap-6 p-4">
				<!-- Section 1: Record payment -->
				<form
					method="POST"
					action="?/recordPayment"
					use:enhance={() => {
						isSubmittingPayment = true;
						return async ({ update }) => {
							await update();
							isSubmittingPayment = false;
						};
					}}
					class="flex flex-col gap-4"
				>
					<input type="hidden" name="lease_id" value={lease.id} />
					<input type="hidden" name="method" value={paymentMethod} />
					<input type="hidden" name="period_month" value={periodMonth} />

					<div class="flex items-center justify-between">
						<h3 class="flex items-center gap-1.5 text-xs font-semibold tracking-wide uppercase">
							<IconCash class="size-4" />
							1. Saisir un règlement (imputation FIFO)
						</h3>
						{#if rentCall?.status === 'PAID'}
							<Badge variant="secondary">Échéance soldée</Badge>
						{/if}
					</div>

					<Field.FieldGroup>
						<div class="grid gap-3 sm:grid-cols-2">
							<Field.Field>
								<Field.Label for="payment-amount">Montant encaissé (€)</Field.Label>
								<Input
									id="payment-amount"
									name="amount_euro"
									type="text"
									inputmode="decimal"
									placeholder="850,00"
									bind:value={amountEuroInput}
									required
								/>
							</Field.Field>

							<Field.Field>
								<Field.Label for="payment-date">Date de règlement</Field.Label>
								<Input
									id="payment-date"
									name="payment_date"
									type="date"
									bind:value={paymentDate}
									required
								/>
							</Field.Field>
						</div>

						<Field.Field>
							<Field.Label>Mode de règlement</Field.Label>
							<Select.Root
								type="single"
								value={paymentMethod}
								onValueChange={(val) => {
									if (val) paymentMethod = val;
								}}
							>
								<Select.Trigger class="w-full">
									{selectedMethodLabel}
								</Select.Trigger>
								<Select.Content>
									<Select.Group>
										{#each PAYMENT_METHODS as method (method.value)}
											<Select.Item value={method.value} label={method.label}>
												{method.label}
											</Select.Item>
										{/each}
									</Select.Group>
								</Select.Content>
							</Select.Root>
						</Field.Field>

						<div class="grid gap-3 sm:grid-cols-2">
							<Field.Field>
								<Field.Label for="payment-label">Libellé comptable</Field.Label>
								<Input
									id="payment-label"
									name="label"
									type="text"
									placeholder="Loyer Octobre 2026"
									bind:value={paymentLabel}
									required
								/>
							</Field.Field>

							<Field.Field>
								<Field.Label for="payment-ref">
									{paymentMethod === 'CASH' ? 'N° reçu espèces' : 'Réf. bancaire / chèque'}
								</Field.Label>
								<Input
									id="payment-ref"
									name="reference"
									type="text"
									placeholder={paymentMethod === 'CASH' ? 'RECU-2026-01' : 'VIR-99412'}
									bind:value={paymentReference}
								/>
							</Field.Field>
						</div>
					</Field.FieldGroup>

					<Button type="submit" disabled={isSubmittingPayment}>
						{#if isSubmittingPayment}
							<Spinner data-icon="inline-start" />
							Imputation en cours...
						{:else}
							Valider le règlement
						{/if}
					</Button>
				</form>

				<Separator />

				<!-- Section 2: Documents (Avis d'échéance & Quittances) -->
				<div class="flex flex-col gap-3">
					<h3 class="flex items-center gap-1.5 text-xs font-semibold tracking-wide uppercase">
						<IconFileText class="size-4" />
						2. Avis d'échéance & Quittances
					</h3>

					{#if templates.length > 0}
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
							class="flex flex-col gap-2 sm:flex-row sm:items-end"
						>
							<input type="hidden" name="lease_id" value={lease.id} />
							<input type="hidden" name="legal_entity_id" value={lease.legal_entity_id} />
							<input type="hidden" name="template_id" value={selectedTemplateId} />
							<input type="hidden" name="tenant_name" value={tenantName} />
							<input type="hidden" name="entity_name" value={entityName} />
							<input type="hidden" name="property_label" value={propertyAndLotLabel} />
							<input type="hidden" name="period_month" value={periodMonth} />
							<input type="hidden" name="amount_due" value={formattedAmountDue} />

							<div class="flex-1">
								<Select.Root
									type="single"
									value={selectedTemplateId}
									onValueChange={(val) => {
										if (val) selectedTemplateId = val;
									}}
								>
									<Select.Trigger class="w-full">
										{selectedTemplateLabel}
									</Select.Trigger>
									<Select.Content>
										<Select.Group>
											{#each templates as tpl (tpl.id)}
												<Select.Item value={tpl.id} label={tpl.name}>
													{tpl.name} ({tpl.template_type})
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
								Générer le document
							</Button>
						</form>
					{:else}
						<p class="text-xs text-muted-foreground">
							Aucun modèle de document enregistré. Vous pouvez en créer un depuis les paramètres ou
							utiliser l'assemblage GLI.
						</p>
					{/if}

					{#if leaseDocuments.length > 0}
						<div class="flex flex-col gap-1.5 rounded-md border p-2.5">
							{#each leaseDocuments as doc (doc.id)}
								<div class="flex items-center justify-between gap-2 text-xs">
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
											Ouvrir
										</Button>
									{/if}
								</div>
							{/each}
						</div>
					{:else}
						<p class="text-xs text-muted-foreground">Aucun document généré pour ce bail.</p>
					{/if}
				</div>

				<Separator />

				<!-- Section 3: Reminders & GLI Claim Dossier -->
				<div class="flex flex-col gap-3">
					<h3 class="flex items-center gap-1.5 text-xs font-semibold tracking-wide uppercase">
						<IconShieldExclamation class="size-4" />
						3. Relances & Dossier Sinistre GLI
					</h3>

					{#if negativePosition && negativePosition.total_unpaid_cents > 0}
						<div
							class="flex flex-col gap-2 rounded-md border border-destructive/30 bg-destructive/10 p-3 text-xs"
						>
							<div class="flex items-center justify-between">
								<span class="font-semibold">Arriéré locatif constaté</span>
								<Badge variant="destructive">
									{formatCentsToEuro(negativePosition.total_unpaid_cents)}
								</Badge>
							</div>
							<p class="text-muted-foreground">
								Mois impayés : {negativePosition.unpaid_months.join(', ')} · Plus ancienne échéance :
								{formatIsoDateFr(negativePosition.oldest_due_date)}
							</p>
							{#if negativePosition.gli_policy_number}
								<p class="font-mono text-[11px]">
									Police GLI : {negativePosition.gli_policy_number}
									{negativePosition.gli_eligible ? '(Éligible déclaration)' : ''}
								</p>
							{/if}
						</div>
					{:else}
						<p class="text-xs text-muted-foreground">
							Aucun arriéré impayé détecté sur ce bail. Vous pouvez néanmoins compiler une archive
							documentaire GLI si nécessaire.
						</p>
					{/if}

					<form
						method="POST"
						action="?/assembleGli"
						use:enhance={() => {
							isAssemblingGli = true;
							return async ({ update }) => {
								await update();
								isAssemblingGli = false;
							};
						}}
					>
						<input type="hidden" name="lease_id" value={lease.id} />
						<input type="hidden" name="template_id" value={gliTemplateId} />
						<Button
							type="submit"
							class="w-full"
							variant={negativePosition && negativePosition.total_unpaid_cents > 0
								? 'destructive'
								: 'outline'}
							disabled={isAssemblingGli}
						>
							{#if isAssemblingGli}
								<Spinner data-icon="inline-start" />
								Assemblage de l'archive .zip...
							{:else}
								<IconFileZip data-icon="inline-start" />
								Assembler le dossier sinistre GLI (.zip)
							{/if}
						</Button>
					</form>

					{#if assembledGliArchive && assembledGliArchive.lease_id === lease.id}
						<div class="flex flex-col gap-2 rounded-md border bg-muted/50 p-3 text-xs">
							<div class="flex items-center justify-between">
								<span class="font-semibold">Archive GLI prête</span>
								<span class="font-mono">
									{formatCentsToEuro(assembledGliArchive.total_unpaid_cents)}
								</span>
							</div>
							<p class="text-[11px] text-muted-foreground">
								Pièces incluses ({assembledGliArchive.included_files.length}) :
								{assembledGliArchive.included_files.join(', ')}
							</p>
							{#if assembledGliArchive.download_url}
								<Button
									size="sm"
									variant="default"
									href={assembledGliArchive.download_url}
									target="_blank"
									rel="noreferrer"
								>
									<IconDownload data-icon="inline-start" />
									Télécharger le dossier .zip
								</Button>
							{/if}
						</div>
					{/if}
				</div>
			</div>
		{/if}
	</Sheet.Content>
</Sheet.Root>
