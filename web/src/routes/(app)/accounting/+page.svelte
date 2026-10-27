<script lang="ts">
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
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import * as Table from '#lib/components/ui/table/index.js';
	import * as Tabs from '#lib/components/ui/tabs/index.js';
	import { ApiError } from '#lib/core/api/api-error';
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
		JournalEntryDTO,
		StatementSection
	} from '#lib/modules/accounting/types';
	import type { LegalEntityDTO } from '#lib/modules/portfolio/types';
	import { apiRegistry } from '#lib/modules/registry';

	let activeTab = $state<'statements' | 'journal'>('statements');
	let fiscalYear = $state<number>(new Date().getFullYear());
	let activeEntityId = $state<string>('');
	let disabledLineIds = $state<string[]>([]);

	let entities = $state<LegalEntityDTO[]>([]);
	let statements = $state<ComputedFinancialStatementsDTO | null>(null);
	let journalEntries = $state<JournalEntryDTO[]>([]);
	let generatedReport = $state<AnnualAccountsReportDTO | null>(null);

	let isLoading = $state(true);
	let isRecomputing = $state(false);
	let isGeneratingPdf = $state(false);
	let isExportingFec = $state(false);
	let errorMessage = $state<string | null>(null);
	let reloadCounter = $state(0);

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

	$effect(() => {
		if (workspaceStore.selectedEntityId !== 'ALL') {
			activeEntityId = workspaceStore.selectedEntityId;
		} else if (entities.length > 0 && !activeEntityId) {
			activeEntityId = entities[0].id;
		}
	});

	$effect(() => {
		const selectedWorkspaceEntity = workspaceStore.selectedEntityId;
		const year = fiscalYear;
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

				const targetEntityId =
					selectedWorkspaceEntity !== 'ALL'
						? selectedWorkspaceEntity
						: activeEntityId || allEntities[0]?.id || '';

				if (!targetEntityId) {
					statements = null;
					journalEntries = [];
					return;
				}
				activeEntityId = targetEntityId;

				const [preview, entries] = await Promise.all([
					apiRegistry.accounting.previewStatements(targetEntityId, year, disabledLineIds),
					apiRegistry.accounting.listJournalEntries(targetEntityId, year)
				]);
				if (cancelled) return;

				statements = preview;
				journalEntries = entries;
			} catch (err) {
				if (cancelled) return;
				errorMessage =
					err instanceof ApiError
						? err.message
						: 'Impossible de charger les états financiers et le journal comptable.';
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

	const activeEntity = $derived(entities.find((e) => e.id === activeEntityId) ?? null);

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

	async function handleToggleStatementLine(lineId: string) {
		if (!activeEntityId) return;
		const nextDisabled = toggleDisabledLineId(disabledLineIds, lineId);
		disabledLineIds = nextDisabled;
		isRecomputing = true;
		try {
			statements = await apiRegistry.accounting.previewStatements(
				activeEntityId,
				fiscalYear,
				nextDisabled
			);
		} catch (err) {
			toast.error(
				err instanceof ApiError ? err.message : 'Erreur lors du recalcul interactif.'
			);
		} finally {
			isRecomputing = false;
		}
	}

	async function handleGeneratePdf() {
		if (!activeEntityId) return;
		isGeneratingPdf = true;
		try {
			const report = await apiRegistry.accounting.generateAnnualAccountsPdf(
				activeEntityId,
				fiscalYear,
				disabledLineIds
			);
			generatedReport = report;
			toast.success(`Plaquette des comptes annuels ${fiscalYear} générée.`);
		} catch (err) {
			toast.error(
				err instanceof ApiError
					? err.message
					: 'Échec de la génération du PDF des comptes annuels.'
			);
		} finally {
			isGeneratingPdf = false;
		}
	}

	async function handleExportFec() {
		if (!activeEntityId) return;
		isExportingFec = true;
		try {
			const tsvContent = await apiRegistry.accounting.exportFecTsv(activeEntityId, fiscalYear);
			const blob = new Blob([tsvContent], { type: 'text/tab-separated-values;charset=utf-8' });
			const url = URL.createObjectURL(blob);
			const anchor = document.createElement('a');
			const siren = (activeEntity?.siret ?? '000000000').slice(0, 9) || '000000000';
			anchor.href = url;
			anchor.download = `${siren}FEC${fiscalYear}1231.tsv`;
			document.body.appendChild(anchor);
			anchor.click();
			document.body.removeChild(anchor);
			URL.revokeObjectURL(url);
			toast.success('Export normatif DGFiP FEC (18 colonnes TSV) téléchargé.');
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Échec de l’export FEC.');
		} finally {
			isExportingFec = false;
		}
	}

	async function handleUpsertBalanceLine(event: SubmitEvent) {
		event.preventDefault();
		if (!activeEntityId || !balAccountCode.trim() || !balLabel.trim()) return;

		let netNCents = 0;
		let netNMinus1Cents = 0;
		try {
			netNCents = parseEuroToCents(balNetNEuro || '0');
			netNMinus1Cents = parseEuroToCents(balNetNMinus1Euro || '0');
		} catch {
			toast.error('Montant N ou N-1 invalide.');
			return;
		}

		const existingLines: AccountBalanceLineDTO[] = (statements?.lines ?? []).map((line) => ({
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

		const lineId = `line_${balAccountCode.trim()}`;
		const filtered = existingLines.filter((l) => l.line_id !== lineId);
		const nextLines: AccountBalanceLineDTO[] = [
			...filtered,
			{
				line_id: lineId,
				account_code: balAccountCode.trim(),
				label: balLabel.trim(),
				section: balSection,
				cerfa_code: balCerfaCode.trim() || undefined,
				gross_n_cents: netNCents,
				depreciation_n_cents: 0,
				net_n_cents: netNCents,
				net_n_minus_1_cents: netNMinus1Cents
			}
		];

		isSavingBalanceLine = true;
		try {
			await apiRegistry.accounting.upsertBalanceLines(activeEntityId, fiscalYear, nextLines);
			toast.success('Ligne de balance enregistrée et états financiers recalculés.');
			balanceSheetOpen = false;
			reloadCounter += 1;
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Erreur lors de la mise à jour.');
		} finally {
			isSavingBalanceLine = false;
		}
	}

	async function handleRecordJournalEntry(event: SubmitEvent) {
		event.preventDefault();
		if (!activeEntityId || !journalBalanceCheck.balanced) {
			toast.error('L’écriture doit être équilibrée (Total Débit = Total Crédit > 0).');
			return;
		}

		isSavingJournal = true;
		try {
			await apiRegistry.accounting.recordJournalEntry(activeEntityId, {
				fiscal_year: fiscalYear,
				journal_code: jrnCode.trim() || 'OD',
				journal_label: jrnLabel.trim() || 'Opérations diverses',
				document_ref: jrnDocRef.trim() || `PIECE-${Date.now()}`,
				label: jrnEntryLabel.trim(),
				lines: [
					{
						line_id: `jrn_${jrnDebitAccount.trim()}`,
						account_code: jrnDebitAccount.trim(),
						account_label: jrnDebitAccountLabel.trim(),
						section: jrnDebitSection,
						label: jrnEntryLabel.trim(),
						debit_cents: journalBalanceCheck.debitCents,
						credit_cents: 0
					},
					{
						line_id: `jrn_${jrnCreditAccount.trim()}`,
						account_code: jrnCreditAccount.trim(),
						account_label: jrnCreditAccountLabel.trim(),
						section: jrnCreditSection,
						label: jrnEntryLabel.trim(),
						debit_cents: 0,
						credit_cents: journalBalanceCheck.creditCents
					}
				]
			});
			toast.success('Écriture comptable en partie double validée.');
			journalSheetOpen = false;
			reloadCounter += 1;
		} catch (err) {
			toast.error(
				err instanceof ApiError ? err.message : 'Impossible d’enregistrer l’écriture comptable.'
			);
		} finally {
			isSavingJournal = false;
		}
	}
</script>

<PageHeader
	title="Comptes Annuels Interactifs, Liasse 2072/2033 & Export FEC"
	subtitle="Simulation en temps réel (activation/désactivation de lignes type amortissements), génération PDF et export DGFiP 18 colonnes"
>
	{#snippet actions()}
		{#if entities.length > 0}
			<Select.Root
				type="single"
				value={activeEntityId}
				onValueChange={(v) => {
					if (v) {
						activeEntityId = v;
						workspaceStore.setEntityId(v);
					}
				}}
			>
				<Select.Trigger class="h-8 w-48">
					{activeEntity?.name ?? 'Sélectionner une entité'}
				</Select.Trigger>
				<Select.Content>
					<Select.Group>
						{#each entities as entity (entity.id)}
							<Select.Item value={entity.id} label={entity.name}>{entity.name}</Select.Item>
						{/each}
					</Select.Group>
				</Select.Content>
			</Select.Root>
		{/if}

		<Input
			type="number"
			min="2000"
			max="2100"
			value={String(fiscalYear)}
			onchange={(e) => {
				const parsed = Number.parseInt((e.currentTarget as HTMLInputElement).value, 10);
				if (!Number.isNaN(parsed)) fiscalYear = parsed;
			}}
			class="h-8 w-24 font-mono"
			aria-label="Exercice fiscal"
		/>

		<Button
			size="sm"
			variant="outline"
			onclick={handleExportFec}
			disabled={isExportingFec || !activeEntityId}
		>
			{#if isExportingFec}
				<Spinner data-icon="inline-start" />
			{:else}
				<IconFileExport data-icon="inline-start" />
			{/if}
			Export FEC (.tsv)
		</Button>

		<Button
			size="sm"
			onclick={handleGeneratePdf}
			disabled={isGeneratingPdf || !activeEntityId}
		>
			{#if isGeneratingPdf}
				<Spinner data-icon="inline-start" />
			{:else}
				<IconFileText data-icon="inline-start" />
			{/if}
			Générer Plaquette PDF
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
				Journal Comptable PCG ({journalEntries.length} écritures)
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
		{#if isLoading}
			<Skeleton class="h-64 w-full" />
		{:else if !statements}
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
						<p class="text-muted-foreground text-xs tabular-nums">
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
						<p class="text-muted-foreground text-xs tabular-nums">
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
						<p class="text-muted-foreground text-xs tabular-nums">
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
						<p class="text-muted-foreground text-xs tabular-nums">
							Capitaux propres & dettes :
							{formatCentsToEuro(statements.total_equity_and_liabilities_n_cents)}
						</p>
					</Card.Content>
				</Card.Root>
			</div>

			<Card.Root>
				<Card.Header class="flex flex-col justify-between gap-2 border-b pb-3 sm:flex-row sm:items-center">
					<div>
						<Card.Title class="text-sm font-semibold">
							Postes du Bilan & Compte de Résultat — Exercice {fiscalYear}
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
						<p class="text-muted-foreground px-4 py-6 text-xs">
							Aucun poste de balance ou écriture comptable sur l'exercice {fiscalYear}. Cliquez sur
							« Ligne de balance » ou « Nouvelle écriture OD/PCG » pour alimenter les états.
						</p>
					{:else}
						<Table.Root>
							<Table.Header>
								<Table.Row>
									<Table.Head class="w-12 pl-4">Actif</Table.Head>
									<Table.Head>Compte PCG</Table.Head>
									<Table.Head>Intitulé du poste</Table.Head>
									<Table.Head>Rubrique & CERFA</Table.Head>
									<Table.Head class="text-right">Net {fiscalYear}</Table.Head>
									<Table.Head class="text-right">Net {fiscalYear - 1}</Table.Head>
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
										<Table.Cell class="text-muted-foreground text-right font-mono tabular-nums">
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
					Livre-Journal PCG (Écritures validées {fiscalYear})
				</Card.Title>
				<Card.Description>
					Toutes les écritures validées ci-dessous sont exportées au format normatif DGFiP FEC (18
					colonnes)
				</Card.Description>
			</Card.Header>
			<Card.Content class="p-0">
				{#if journalEntries.length === 0}
					<p class="text-muted-foreground px-4 py-6 text-xs">
						Aucune écriture comptable enregistrée pour l'exercice {fiscalYear}.
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
							{#each journalEntries as entry (entry.id)}
								<Table.Row>
									<Table.Cell class="pl-4 text-xs">
										<span class="font-medium">{formatIsoDateFr(entry.entry_date)}</span>
										<span class="text-muted-foreground block font-mono text-[11px]">
											{entry.entry_number}
										</span>
									</Table.Cell>
									<Table.Cell>
										<Badge variant="outline">{entry.journal_code}</Badge>
									</Table.Cell>
									<Table.Cell class="font-mono text-xs">{entry.document_ref}</Table.Cell>
									<Table.Cell class="text-xs">
										<span class="font-semibold">{entry.label}</span>
										<div class="text-muted-foreground mt-1 flex flex-col gap-0.5 font-mono text-[11px]">
											{#each entry.lines as line (line.id)}
												<span>
													{line.account_code} {line.account_label} :
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
				l'exercice {fiscalYear}.
			</Sheet.Description>
		</Sheet.Header>

		<form onsubmit={handleUpsertBalanceLine} class="flex flex-col gap-4 p-4">
			<Field.FieldGroup>
				<div class="grid gap-3 sm:grid-cols-2">
					<Field.Field>
						<Field.Label for="bal-code">N° de compte PCG</Field.Label>
						<Input id="bal-code" class="font-mono" bind:value={balAccountCode} required />
					</Field.Field>

					<Field.Field>
						<Field.Label for="bal-cerfa">Case CERFA (2072 / 2033)</Field.Label>
						<Input id="bal-cerfa" placeholder="2072-R1 / 2033-A" bind:value={balCerfaCode} />
					</Field.Field>
				</div>

				<Field.Field>
					<Field.Label for="bal-label">Intitulé du compte</Field.Label>
					<Input id="bal-label" bind:value={balLabel} required />
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
						<Field.Label for="bal-n">Montant Net {fiscalYear} (€)</Field.Label>
						<Input
							id="bal-n"
							type="text"
							inputmode="decimal"
							bind:value={balNetNEuro}
							required
						/>
					</Field.Field>

					<Field.Field>
						<Field.Label for="bal-n1">Montant Net {fiscalYear - 1} (€)</Field.Label>
						<Input
							id="bal-n1"
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

		<form onsubmit={handleRecordJournalEntry} class="flex flex-col gap-4 p-4">
			<Field.FieldGroup>
				<div class="grid gap-3 sm:grid-cols-3">
					<Field.Field>
						<Field.Label for="jrn-code">Code journal</Field.Label>
						<Input id="jrn-code" bind:value={jrnCode} required />
					</Field.Field>

					<Field.Field>
						<Field.Label for="jrn-lbl">Libellé journal</Field.Label>
						<Input id="jrn-lbl" bind:value={jrnLabel} required />
					</Field.Field>

					<Field.Field>
						<Field.Label for="jrn-ref">Réf. pièce</Field.Label>
						<Input id="jrn-ref" bind:value={jrnDocRef} required />
					</Field.Field>
				</div>

				<Field.Field>
					<Field.Label for="jrn-entry-lbl">Libellé de l'écriture</Field.Label>
					<Input id="jrn-entry-lbl" bind:value={jrnEntryLabel} required />
				</Field.Field>

				<div class="rounded-md border p-3">
					<span class="text-xs font-semibold uppercase">Ligne au Débit</span>
					<div class="mt-2 grid gap-2 sm:grid-cols-3">
						<Input placeholder="Compte (681100)" bind:value={jrnDebitAccount} required />
						<Input placeholder="Libellé compte" bind:value={jrnDebitAccountLabel} required />
						<Input
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
						<Input placeholder="Compte (281300)" bind:value={jrnCreditAccount} required />
						<Input placeholder="Libellé compte" bind:value={jrnCreditAccountLabel} required />
						<Input
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
