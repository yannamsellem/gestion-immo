<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import IconAlertTriangle from '@tabler/icons-svelte/icons/alert-triangle';
	import IconBook from '@tabler/icons-svelte/icons/book';
	import IconCalculator from '@tabler/icons-svelte/icons/calculator';
	import IconDownload from '@tabler/icons-svelte/icons/download';
	import IconFileExport from '@tabler/icons-svelte/icons/file-export';
	import IconFileText from '@tabler/icons-svelte/icons/file-text';
	import IconPlus from '@tabler/icons-svelte/icons/plus';
	import { toast } from 'svelte-sonner';
	import PageHeader from '#lib/components/layout/PageHeader.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { Checkbox } from '#lib/components/ui/checkbox/index.js';
	import * as Empty from '#lib/components/ui/empty/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import * as Select from '#lib/components/ui/select/index.js';
	import * as Sheet from '#lib/components/ui/sheet/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import * as Table from '#lib/components/ui/table/index.js';
	import * as Tabs from '#lib/components/ui/tabs/index.js';
	import { formatIsoDateFr } from '#lib/core/formatters/date';
	import {
		formatBasisPointsToPercent,
		formatCentsToEuro,
		parseEuroToCents
	} from '#lib/core/formatters/money';
	import { workspaceStore } from '#lib/core/workspace/workspace.svelte';
	import {
		STATEMENT_SECTION_OPTIONS,
		formatStatementSectionLabel,
		toggleDisabledLineId
	} from '#lib/modules/accounting/statement-state';
	import type {
		AccountBalanceLineDTO,
		AnnualAccountsReportDTO,
		ComputedFinancialStatementsDTO,
		StatementSection
	} from '#lib/modules/accounting/types';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	let activeTab = $state<'statements' | 'journal'>('statements');
	let disabledLineIds = $state<string[]>([]);
	let localStatements = $state<ComputedFinancialStatementsDTO | null>(null);
	let generatedReport = $state<AnnualAccountsReportDTO | null>(null);

	let isRecomputing = $state(false);
	let isGeneratingPdf = $state(false);

	// Upsert Balance Line Sheet
	let balanceSheetOpen = $state(false);
	let balAccountCode = $state('706000');
	let balLabel = $state('Loyers et revenus locatifs');
	let balSection = $state<StatementSection>('OPERATING_REVENUE');
	let balCerfaCode = $state('2072-R1');
	let balNetNEuro = $state('18000,00');
	let balNetNMinus1Euro = $state('16500,00');
	let isSavingBalanceLine = $state(false);

	// Record Double-Entry Journal Entry Sheet
	let journalSheetOpen = $state(false);
	let jrnCode = $state('OD');
	let jrnLabel = $state('Opérations diverses');
	let jrnDocRef = $state('FACT-2026-01');
	let jrnEntryLabel = $state('Dotation aux amortissements / Charge locative');
	let jrnDebitAccount = $state('681100');
	let jrnDebitAccountLabel = $state('Dotations aux amortissements sur immobilisations');
	let jrnDebitSection = $state<StatementSection>('DEPRECIATION');
	let jrnDebitEuro = $state('1200,00');
	let jrnCreditAccount = $state('281300');
	let jrnCreditAccountLabel = $state('Amortissements des constructions');
	let jrnCreditSection = $state<StatementSection>('FIXED_ASSETS');
	let jrnCreditEuro = $state('1200,00');
	let isSavingJournal = $state(false);

	let recomputeFormEl = $state<HTMLFormElement | null>(null);

	$effect(() => {
		localStatements = data.statements;
		disabledLineIds = data.disabledLineIds;
	});

	$effect(() => {
		if (form?.successMessage) {
			toast.success(form.successMessage);
		}
		if (form?.actionError) {
			toast.error(form.actionError);
		}
		if (form?.recomputedStatements) {
			localStatements = form.recomputedStatements;
			disabledLineIds = form.disabledLineIds ?? [];
		}
		if (form?.generatedReport) {
			generatedReport = form.generatedReport;
		}
		if (form?.balanceLineSaved) {
			balanceSheetOpen = false;
		}
		if (form?.journalRecorded) {
			journalSheetOpen = false;
		}
	});

	const statements = $derived(localStatements ?? data.statements);
	const activeEntity = $derived(data.entities.find((e) => e.id === data.activeEntityId) ?? null);

	const existingBalanceLinesJson = $derived.by(() => {
		const lines: AccountBalanceLineDTO[] = (statements?.lines ?? []).map((line) => ({
			line_id: line.line_id,
			account_code: line.account_code,
			label: line.label,
			section: line.section,
			cerfa_code: line.cerfa_code,
			gross_n_cents: line.gross_n_cents,
			depreciation_n_cents: line.depreciation_n_cents,
			net_n_cents: line.net_n_cents,
			net_n_minus_1_cents: line.net_n_minus_1_cents
		}));
		return JSON.stringify(lines);
	});

	const fecExportHref = $derived.by(() => {
		if (!data.activeEntityId) return '';
		const siren = (activeEntity?.siret ?? '000000000').slice(0, 9) || '000000000';
		return `${resolve('/(app)/accounting')}/fec?entityId=${encodeURIComponent(data.activeEntityId)}&fiscalYear=${data.fiscalYear}&siren=${encodeURIComponent(siren)}`;
	});

	const journalBalanceCheck = $derived.by(() => {
		try {
			const debitCents = parseEuroToCents(jrnDebitEuro || '0');
			const creditCents = parseEuroToCents(jrnCreditEuro || '0');
			return {
				debitCents,
				creditCents,
				balanced: debitCents > 0 && debitCents === creditCents
			};
		} catch {
			return { debitCents: 0, creditCents: 0, balanced: false };
		}
	});

	function handleToggleStatementLine(lineId: string) {
		disabledLineIds = toggleDisabledLineId(disabledLineIds, lineId);
		queueMicrotask(() => {
			recomputeFormEl?.requestSubmit();
		});
	}
</script>

<!-- Hidden form for interactive SSR statement line toggle -->
<form
	bind:this={recomputeFormEl}
	method="POST"
	action="?/recompute"
	use:enhance={() => {
		isRecomputing = true;
		return async ({ update }) => {
			await update({ reset: false });
			isRecomputing = false;
		};
	}}
	class="hidden"
>
	<input type="hidden" name="entity_id" value={data.activeEntityId} />
	<input type="hidden" name="fiscal_year" value={data.fiscalYear} />
	<input type="hidden" name="disabled_line_ids_json" value={JSON.stringify(disabledLineIds)} />
</form>

<PageHeader
	title="Comptes Annuels Interactifs, Liasse 2072/2033 & Export FEC"
	subtitle="Simulation en temps réel (activation/désactivation de lignes type amortissements), génération PDF et export DGFiP 18 colonnes"
>
	{#snippet actions()}
		{#if data.entities.length > 0}
			<Select.Root
				type="single"
				value={data.activeEntityId}
				onValueChange={(v) => {
					if (v) {
						workspaceStore.setEntityId(v);
					}
				}}
			>
				<Select.Trigger class="h-8 w-48">
					{activeEntity?.name ?? 'Sélectionner une entité'}
				</Select.Trigger>
				<Select.Content>
					<Select.Group>
						{#each data.entities as entity (entity.id)}
							<Select.Item value={entity.id} label={entity.name}>{entity.name}</Select.Item>
						{/each}
					</Select.Group>
				</Select.Content>
			</Select.Root>
		{/if}

		<form method="GET">
			<Input
				type="number"
				name="year"
				min="2000"
				max="2100"
				value={String(data.fiscalYear)}
				onchange={(e) => {
					const input = e.currentTarget as HTMLInputElement;
					const parsed = Number.parseInt(input.value, 10);
					if (!Number.isNaN(parsed)) {
						input.form?.requestSubmit();
					}
				}}
				class="h-8 w-24 font-mono"
				aria-label="Exercice fiscal"
			/>
		</form>

		{#if fecExportHref}
			<Button size="sm" variant="outline" href={fecExportHref}>
				<IconFileExport data-icon="inline-start" />
				Export FEC (.tsv)
			</Button>
		{/if}

		<form
			method="POST"
			action="?/generatePdf"
			use:enhance={() => {
				isGeneratingPdf = true;
				return async ({ update }) => {
					await update({ reset: false });
					isGeneratingPdf = false;
				};
			}}
		>
			<input type="hidden" name="entity_id" value={data.activeEntityId} />
			<input type="hidden" name="fiscal_year" value={data.fiscalYear} />
			<input type="hidden" name="disabled_line_ids_json" value={JSON.stringify(disabledLineIds)} />
			<Button type="submit" size="sm" disabled={isGeneratingPdf || !data.activeEntityId}>
				{#if isGeneratingPdf}
					<Spinner data-icon="inline-start" />
				{:else}
					<IconFileText data-icon="inline-start" />
				{/if}
				Générer Plaquette PDF
			</Button>
		</form>
	{/snippet}
</PageHeader>

{#if data.errorMessage}
	<Alert.Root variant="destructive">
		<IconAlertTriangle class="size-4" />
		<Alert.Title>Erreur</Alert.Title>
		<Alert.Description>{data.errorMessage}</Alert.Description>
	</Alert.Root>
{/if}

{#if generatedReport}
	<Card.Root>
		<Card.Content class="flex flex-wrap items-center justify-between gap-3 py-3">
			<div class="flex flex-col text-xs">
				<span class="font-semibold">
					Plaquette générée : {generatedReport.file_name} (Résultat Net :
					{formatCentsToEuro(generatedReport.net_income_n_cents)})
				</span>
				<span class="text-muted-foreground">
					{generatedReport.disabled_line_ids.length} ligne(s) neutralisée(s) hors simulation
				</span>
			</div>
			{#if generatedReport.download_url}
				<Button
					size="sm"
					variant="secondary"
					href={generatedReport.download_url}
					target="_blank"
					rel="noreferrer"
				>
					<IconDownload data-icon="inline-start" />
					Ouvrir le PDF
				</Button>
			{/if}
		</Card.Content>
	</Card.Root>
{/if}

<Tabs.Root bind:value={activeTab}>
	<div class="flex flex-wrap items-center justify-between gap-3">
		<Tabs.List>
			<Tabs.Trigger value="statements">
				Bilan, Compte de Résultat & SIG ({statements?.lines.length ?? 0} postes)
			</Tabs.Trigger>
			<Tabs.Trigger value="journal">
				Journal Comptable PCG ({data.journalEntries.length} écritures)
			</Tabs.Trigger>
		</Tabs.List>

		<div class="flex items-center gap-2">
			<Button size="sm" variant="outline" onclick={() => (balanceSheetOpen = true)}>
				<IconPlus data-icon="inline-start" />
				Ligne de balance
			</Button>
			<Button size="sm" variant="secondary" onclick={() => (journalSheetOpen = true)}>
				<IconBook data-icon="inline-start" />
				Nouvelle écriture OD/PCG
			</Button>
		</div>
	</div>

	<!-- Tab 1: Interactive Financial Statements -->
	<Tabs.Content value="statements" class="mt-4 flex flex-col gap-6">
		{#if !statements}
			<Card.Root>
				<Card.Content class="py-10">
					<Empty.Root>
						<Empty.Header>
							<Empty.Media variant="icon">
								<IconCalculator class="size-5" />
							</Empty.Media>
							<Empty.Title>Aucune donnée comptable disponible</Empty.Title>
							<Empty.Description>
								Sélectionnez ou créez une entité juridique pour visualiser son bilan, son compte de
								résultat et sa liasse fiscale.
							</Empty.Description>
						</Empty.Header>
					</Empty.Root>
				</Card.Content>
			</Card.Root>
		{:else}
			<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
				<Card.Root>
					<Card.Header class="pb-2">
						<Card.Description>Produits d'exploitation (N)</Card.Description>
						<Card.Title class="text-xl font-semibold tabular-nums">
							{formatCentsToEuro(statements.total_operating_revenue_n_cents)}
						</Card.Title>
					</Card.Header>
					<Card.Content>
						<p class="text-xs text-muted-foreground tabular-nums">
							N-1 : {formatCentsToEuro(statements.total_operating_revenue_n_minus_1_cents)} · EBE :
							{formatCentsToEuro(statements.gross_operating_surplus_n_cents)}
						</p>
					</Card.Content>
				</Card.Root>

				<Card.Root>
					<Card.Header class="pb-2">
						<Card.Description>Total des charges (N)</Card.Description>
						<Card.Title class="text-xl font-semibold tabular-nums">
							{formatCentsToEuro(statements.total_expenses_n_cents)}
						</Card.Title>
					</Card.Header>
					<Card.Content>
						<p class="text-xs text-muted-foreground tabular-nums">
							Dont dotations amort. : {formatCentsToEuro(statements.total_depreciation_n_cents)}
						</p>
					</Card.Content>
				</Card.Root>

				<Card.Root>
					<Card.Header class="pb-2">
						<Card.Description>Résultat Net Comptable (N)</Card.Description>
						<Card.Title class="text-xl font-semibold tabular-nums">
							{formatCentsToEuro(statements.net_income_n_cents)}
						</Card.Title>
					</Card.Header>
					<Card.Content>
						<p class="text-xs text-muted-foreground tabular-nums">
							Capacité d'Autofinancement (CAF) :
							{formatCentsToEuro(statements.self_financing_capacity_n_cents)}
						</p>
					</Card.Content>
				</Card.Root>

				<Card.Root>
					<Card.Header class="pb-2">
						<Card.Description>Total Bilan Actif / Passif</Card.Description>
						<Card.Title class="text-xl font-semibold tabular-nums">
							{formatCentsToEuro(statements.total_assets_n_cents)}
						</Card.Title>
					</Card.Header>
					<Card.Content>
						<p class="text-xs text-muted-foreground tabular-nums">
							Capitaux propres & dettes :
							{formatCentsToEuro(statements.total_equity_and_liabilities_n_cents)}
						</p>
					</Card.Content>
				</Card.Root>
			</div>

			<Card.Root>
				<Card.Header
					class="flex flex-col justify-between gap-2 border-b pb-3 sm:flex-row sm:items-center"
				>
					<div>
						<Card.Title class="text-sm font-semibold">
							Postes du Bilan & Compte de Résultat — Exercice {data.fiscalYear}
						</Card.Title>
						<Card.Description>
							Décochez une ligne (ex: Dotations aux amortissements 6811) pour recalculer
							instantanément le résultat net et la CAF avant génération PDF.
						</Card.Description>
					</div>
					<div class="flex items-center gap-2">
						{#if isRecomputing}
							<Spinner class="size-4" />
						{/if}
						<Badge variant="outline">
							{disabledLineIds.length} ligne(s) neutralisée(s)
						</Badge>
					</div>
				</Card.Header>

				<Card.Content class="p-0">
					{#if statements.lines.length === 0}
						<p class="px-4 py-6 text-xs text-muted-foreground">
							Aucun poste de balance ou écriture comptable sur l'exercice {data.fiscalYear}. Cliquez
							sur « Ligne de balance » ou « Nouvelle écriture OD/PCG » pour alimenter les états.
						</p>
					{:else}
						<Table.Root>
							<Table.Header>
								<Table.Row>
									<Table.Head class="w-12 pl-4">Actif</Table.Head>
									<Table.Head>Compte PCG</Table.Head>
									<Table.Head>Intitulé du poste</Table.Head>
									<Table.Head>Rubrique & CERFA</Table.Head>
									<Table.Head class="text-right">Net {data.fiscalYear}</Table.Head>
									<Table.Head class="text-right">Net {data.fiscalYear - 1}</Table.Head>
									<Table.Head class="pr-4 text-right">Variation</Table.Head>
								</Table.Row>
							</Table.Header>
							<Table.Body>
								{#each statements.lines as line (line.line_id)}
									{@const isIncluded = !disabledLineIds.includes(line.line_id)}
									<Table.Row class={isIncluded ? '' : 'opacity-50'}>
										<Table.Cell class="pl-4">
											<Checkbox
												checked={isIncluded}
												onCheckedChange={() => handleToggleStatementLine(line.line_id)}
												aria-label={`Inclure le compte ${line.account_code}`}
											/>
										</Table.Cell>
										<Table.Cell class="font-mono text-xs font-semibold">
											{line.account_code}
										</Table.Cell>
										<Table.Cell class="font-medium">{line.label}</Table.Cell>
										<Table.Cell>
											<div class="flex flex-wrap items-center gap-1.5">
												<Badge variant="outline">
													{formatStatementSectionLabel(line.section)}
												</Badge>
												{#if line.cerfa_code}
													<Badge variant="secondary">{line.cerfa_code}</Badge>
												{/if}
											</div>
										</Table.Cell>
										<Table.Cell class="text-right font-mono tabular-nums">
											{formatCentsToEuro(line.net_n_cents)}
										</Table.Cell>
										<Table.Cell class="text-right font-mono text-muted-foreground tabular-nums">
											{formatCentsToEuro(line.net_n_minus_1_cents)}
										</Table.Cell>
										<Table.Cell class="pr-4 text-right font-mono text-xs tabular-nums">
											{#if line.variation_not_significant}
												<span class="text-muted-foreground">NS</span>
											{:else}
												{formatBasisPointsToPercent(line.variation_pct_basis_points)}
											{/if}
										</Table.Cell>
									</Table.Row>
								{/each}
							</Table.Body>
						</Table.Root>
					{/if}
				</Card.Content>
			</Card.Root>
		{/if}
	</Tabs.Content>

	<!-- Tab 2: PCG Double-Entry Journal -->
	<Tabs.Content value="journal" class="mt-4">
		<Card.Root>
			<Card.Header class="border-b pb-3">
				<Card.Title class="text-sm font-semibold">
					Livre-Journal PCG (Écritures validées {data.fiscalYear})
				</Card.Title>
				<Card.Description>
					Toutes les écritures validées ci-dessous sont exportées au format normatif DGFiP FEC (18
					colonnes)
				</Card.Description>
			</Card.Header>
			<Card.Content class="p-0">
				{#if data.journalEntries.length === 0}
					<p class="px-4 py-6 text-xs text-muted-foreground">
						Aucune écriture comptable enregistrée pour l'exercice {data.fiscalYear}.
					</p>
				{:else}
					<Table.Root>
						<Table.Header>
							<Table.Row>
								<Table.Head class="pl-4">Date & N°</Table.Head>
								<Table.Head>Journal</Table.Head>
								<Table.Head>Pièce</Table.Head>
								<Table.Head>Libellé & Lignes PCG</Table.Head>
								<Table.Head class="text-right">Total Débit</Table.Head>
								<Table.Head class="pr-4 text-right">Total Crédit</Table.Head>
							</Table.Row>
						</Table.Header>
						<Table.Body>
							{#each data.journalEntries as entry (entry.id)}
								<Table.Row>
									<Table.Cell class="pl-4 text-xs">
										<span class="font-medium">{formatIsoDateFr(entry.entry_date)}</span>
										<span class="block font-mono text-[11px] text-muted-foreground">
											{entry.entry_number}
										</span>
									</Table.Cell>
									<Table.Cell>
										<Badge variant="outline">{entry.journal_code}</Badge>
									</Table.Cell>
									<Table.Cell class="font-mono text-xs">{entry.document_ref}</Table.Cell>
									<Table.Cell class="text-xs">
										<span class="font-semibold">{entry.label}</span>
										<div
											class="mt-1 flex flex-col gap-0.5 font-mono text-[11px] text-muted-foreground"
										>
											{#each entry.lines as line (line.id)}
												<span>
													{line.account_code}
													{line.account_label} :
													{line.debit_cents > 0
														? `D ${formatCentsToEuro(line.debit_cents)}`
														: `C ${formatCentsToEuro(line.credit_cents)}`}
												</span>
											{/each}
										</div>
									</Table.Cell>
									<Table.Cell class="text-right font-mono tabular-nums">
										{formatCentsToEuro(entry.total_debit_cents)}
									</Table.Cell>
									<Table.Cell class="pr-4 text-right font-mono tabular-nums">
										{formatCentsToEuro(entry.total_credit_cents)}
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

<!-- Upsert Balance Line Sheet -->
<Sheet.Root bind:open={balanceSheetOpen}>
	<Sheet.Content side="right" class="w-full overflow-y-auto sm:max-w-md">
		<Sheet.Header class="border-b pb-4">
			<Sheet.Title>Ajouter / modifier un poste de balance</Sheet.Title>
			<Sheet.Description>
				Alimentez un compte PCG (ex: 706000 Loyers, 681100 Amortissements, 213000 Immeubles) pour
				l'exercice {data.fiscalYear}.
			</Sheet.Description>
		</Sheet.Header>

		<form
			method="POST"
			action="?/upsertBalanceLine"
			use:enhance={() => {
				isSavingBalanceLine = true;
				return async ({ update }) => {
					await update();
					isSavingBalanceLine = false;
				};
			}}
			class="flex flex-col gap-4 p-4"
		>
			<input type="hidden" name="entity_id" value={data.activeEntityId} />
			<input type="hidden" name="fiscal_year" value={data.fiscalYear} />
			<input type="hidden" name="existing_lines_json" value={existingBalanceLinesJson} />
			<input type="hidden" name="section" value={balSection} />

			<Field.FieldGroup>
				<div class="grid gap-3 sm:grid-cols-2">
					<Field.Field>
						<Field.Label for="bal-code">N° de compte PCG</Field.Label>
						<Input
							id="bal-code"
							name="account_code"
							class="font-mono"
							bind:value={balAccountCode}
							required
						/>
					</Field.Field>

					<Field.Field>
						<Field.Label for="bal-cerfa">Case CERFA (2072 / 2033)</Field.Label>
						<Input
							id="bal-cerfa"
							name="cerfa_code"
							placeholder="2072-R1 / 2033-A"
							bind:value={balCerfaCode}
						/>
					</Field.Field>
				</div>

				<Field.Field>
					<Field.Label for="bal-label">Intitulé du compte</Field.Label>
					<Input id="bal-label" name="label" bind:value={balLabel} required />
				</Field.Field>

				<Field.Field>
					<Field.Label>Rubrique des états financiers</Field.Label>
					<Select.Root
						type="single"
						value={balSection}
						onValueChange={(v) => {
							if (v) balSection = v;
						}}
					>
						<Select.Trigger class="w-full">
							{formatStatementSectionLabel(balSection)}
						</Select.Trigger>
						<Select.Content>
							<Select.Group>
								{#each STATEMENT_SECTION_OPTIONS as opt (opt.value)}
									<Select.Item value={opt.value} label={opt.label}>{opt.label}</Select.Item>
								{/each}
							</Select.Group>
						</Select.Content>
					</Select.Root>
				</Field.Field>

				<div class="grid gap-3 sm:grid-cols-2">
					<Field.Field>
						<Field.Label for="bal-n">Montant Net {data.fiscalYear} (€)</Field.Label>
						<Input
							id="bal-n"
							name="net_n_euro"
							type="text"
							inputmode="decimal"
							bind:value={balNetNEuro}
							required
						/>
					</Field.Field>

					<Field.Field>
						<Field.Label for="bal-n1">Montant Net {data.fiscalYear - 1} (€)</Field.Label>
						<Input
							id="bal-n1"
							name="net_n_minus_1_euro"
							type="text"
							inputmode="decimal"
							bind:value={balNetNMinus1Euro}
							required
						/>
					</Field.Field>
				</div>
			</Field.FieldGroup>

			<Button type="submit" disabled={isSavingBalanceLine}>
				{#if isSavingBalanceLine}
					<Spinner data-icon="inline-start" />
				{/if}
				Enregistrer le poste
			</Button>
		</form>
	</Sheet.Content>
</Sheet.Root>

<!-- Record Double-Entry PCG Journal Entry Sheet -->
<Sheet.Root bind:open={journalSheetOpen}>
	<Sheet.Content side="right" class="w-full overflow-y-auto sm:max-w-lg">
		<Sheet.Header class="border-b pb-4">
			<Sheet.Title>Saisir une écriture comptable en partie double</Sheet.Title>
			<Sheet.Description>
				L'équilibre strict Débit = Crédit est vérifié avant l'inscription au livre-journal FEC.
			</Sheet.Description>
		</Sheet.Header>

		<form
			method="POST"
			action="?/recordJournalEntry"
			use:enhance={() => {
				isSavingJournal = true;
				return async ({ update }) => {
					await update();
					isSavingJournal = false;
				};
			}}
			class="flex flex-col gap-4 p-4"
		>
			<input type="hidden" name="entity_id" value={data.activeEntityId} />
			<input type="hidden" name="fiscal_year" value={data.fiscalYear} />
			<input type="hidden" name="debit_section" value={jrnDebitSection} />
			<input type="hidden" name="credit_section" value={jrnCreditSection} />

			<Field.FieldGroup>
				<div class="grid gap-3 sm:grid-cols-3">
					<Field.Field>
						<Field.Label for="jrn-code">Code journal</Field.Label>
						<Input id="jrn-code" name="journal_code" bind:value={jrnCode} required />
					</Field.Field>

					<Field.Field>
						<Field.Label for="jrn-lbl">Libellé journal</Field.Label>
						<Input id="jrn-lbl" name="journal_label" bind:value={jrnLabel} required />
					</Field.Field>

					<Field.Field>
						<Field.Label for="jrn-ref">Réf. pièce</Field.Label>
						<Input id="jrn-ref" name="document_ref" bind:value={jrnDocRef} required />
					</Field.Field>
				</div>

				<Field.Field>
					<Field.Label for="jrn-entry-lbl">Libellé de l'écriture</Field.Label>
					<Input id="jrn-entry-lbl" name="entry_label" bind:value={jrnEntryLabel} required />
				</Field.Field>

				<div class="rounded-md border p-3">
					<span class="text-xs font-semibold uppercase">Ligne au Débit</span>
					<div class="mt-2 grid gap-2 sm:grid-cols-3">
						<Input
							name="debit_account"
							placeholder="Compte (681100)"
							bind:value={jrnDebitAccount}
							required
						/>
						<Input
							name="debit_account_label"
							placeholder="Libellé compte"
							bind:value={jrnDebitAccountLabel}
							required
						/>
						<Input
							name="debit_euro"
							placeholder="Montant €"
							inputmode="decimal"
							bind:value={jrnDebitEuro}
							required
						/>
					</div>
				</div>

				<div class="rounded-md border p-3">
					<span class="text-xs font-semibold uppercase">Ligne au Crédit</span>
					<div class="mt-2 grid gap-2 sm:grid-cols-3">
						<Input
							name="credit_account"
							placeholder="Compte (281300)"
							bind:value={jrnCreditAccount}
							required
						/>
						<Input
							name="credit_account_label"
							placeholder="Libellé compte"
							bind:value={jrnCreditAccountLabel}
							required
						/>
						<Input
							name="credit_euro"
							placeholder="Montant €"
							inputmode="decimal"
							bind:value={jrnCreditEuro}
							required
						/>
					</div>
				</div>
			</Field.FieldGroup>

			<div class="flex items-center justify-between text-xs">
				<span>
					Débit : <strong>{formatCentsToEuro(journalBalanceCheck.debitCents)}</strong> · Crédit :
					<strong>{formatCentsToEuro(journalBalanceCheck.creditCents)}</strong>
				</span>
				<Badge variant={journalBalanceCheck.balanced ? 'secondary' : 'destructive'}>
					{journalBalanceCheck.balanced ? 'Équilibrée' : 'Déséquilibrée'}
				</Badge>
			</div>

			<Button type="submit" disabled={isSavingJournal || !journalBalanceCheck.balanced}>
				{#if isSavingJournal}
					<Spinner data-icon="inline-start" />
				{/if}
				Valider l'écriture au journal
			</Button>
		</form>
	</Sheet.Content>
</Sheet.Root>
