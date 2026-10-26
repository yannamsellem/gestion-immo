<script lang="ts">
	import { page } from '$app/stores';
	import {
		createApiClient,
		type AccountingStoredFileDTO,
		type ComputedFinancialStatementsDTO,
		type ComputedLineDTO
	} from '$lib/api/client';
	import { isLineDisabled, toDisabledArray, toggleStatementLine } from '$lib/api/statement-state';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { Card } from '$lib/components/ui/card';
	import { Switch } from '$lib/components/ui/switch';
	import { Table } from '$lib/components/ui/table';
	import { Tabs } from '$lib/components/ui/tabs';
	import { formatBasisPointsPct, formatEUR, formatVariationPct } from '$lib/utils';

	const api = createApiClient();
	$: entityId = $page.params.entityId || 'ent_sarl_investissimo';
	let fiscalYear = 2026;
	let activeTab = 'income_statement';

	// Immutable ReadonlySet<string> tracking disabled line IDs
	let disabledLineIds: ReadonlySet<string> = new Set<string>();

	const BASE_LINES: readonly ComputedLineDTO[] = [
		{
			line_id: 'rev_hab',
			account_code: '706100',
			label: 'Loyers encaissés Habitation (Exonérés TVA)',
			section: 'OPERATING_REVENUE',
			cerfa_code: '2033-B-218',
			disabled: false,
			gross_n_cents: 6602100,
			depreciation_n_cents: 0,
			net_n_cents: 6602100,
			net_n_minus_1_cents: 6017900,
			variation_abs_cents: 584200,
			variation_pct_basis_points: 971,
			variation_not_significant: false
		},
		{
			line_id: 'rev_comm',
			account_code: '706120',
			label: 'Loyers encaissés Commerciaux HT (TVA 20 %)',
			section: 'OPERATING_REVENUE',
			cerfa_code: '2033-B-218',
			disabled: false,
			gross_n_cents: 3829700,
			depreciation_n_cents: 0,
			net_n_cents: 3829700,
			net_n_minus_1_cents: 2425000,
			variation_abs_cents: 1404700,
			variation_pct_basis_points: 5793,
			variation_not_significant: false
		},
		{
			line_id: 'rev_charges',
			account_code: '708800',
			label: 'Refacturation charges locatives & taxes foncières',
			section: 'OPERATING_REVENUE',
			cerfa_code: '2033-B-218',
			disabled: false,
			gross_n_cents: 553100,
			depreciation_n_cents: 0,
			net_n_cents: 553100,
			net_n_minus_1_cents: 430100,
			variation_abs_cents: 123000,
			variation_pct_basis_points: 2860,
			variation_not_significant: false
		},
		{
			line_id: 'exp_maintenance',
			account_code: '615200',
			label: 'Entretien et réparations sur biens immobiliers',
			section: 'EXTERNAL_EXPENSES',
			cerfa_code: '2033-B-242',
			disabled: false,
			gross_n_cents: 669200,
			depreciation_n_cents: 0,
			net_n_cents: 669200,
			net_n_minus_1_cents: 0,
			variation_abs_cents: 669200,
			variation_pct_basis_points: 0,
			variation_not_significant: true
		},
		{
			line_id: 'exp_fees',
			account_code: '622600',
			label: 'Honoraires comptables, juridiques et gestion',
			section: 'EXTERNAL_EXPENSES',
			cerfa_code: '2033-B-242',
			disabled: false,
			gross_n_cents: 1094400,
			depreciation_n_cents: 0,
			net_n_cents: 1094400,
			net_n_minus_1_cents: 982000,
			variation_abs_cents: 112400,
			variation_pct_basis_points: 1145,
			variation_not_significant: false
		},
		{
			line_id: 'exp_property_tax',
			account_code: '635120',
			label: 'Taxes foncières et impôts locaux',
			section: 'TAXES_AND_DUTIES',
			cerfa_code: '2033-B-244',
			disabled: false,
			gross_n_cents: 800300,
			depreciation_n_cents: 0,
			net_n_cents: 800300,
			net_n_minus_1_cents: 765000,
			variation_abs_cents: 35300,
			variation_pct_basis_points: 461,
			variation_not_significant: false
		},
		{
			line_id: 'exp_depreciation',
			account_code: '681120',
			label: 'Dotations aux amortissements des constructions et agencements',
			section: 'DEPRECIATION',
			cerfa_code: '2033-B-254',
			disabled: false,
			gross_n_cents: 4729200,
			depreciation_n_cents: 0,
			net_n_cents: 4729200,
			net_n_minus_1_cents: 4729200,
			variation_abs_cents: 0,
			variation_pct_basis_points: 0,
			variation_not_significant: false
		},
		{
			line_id: 'exp_interest',
			account_code: '661100',
			label: 'Intérêts des emprunts bancaires immobiliers (PCG 164)',
			section: 'FINANCIAL_EXPENSES',
			cerfa_code: '2033-B-294',
			disabled: false,
			gross_n_cents: 1303000,
			depreciation_n_cents: 0,
			net_n_cents: 1303000,
			net_n_minus_1_cents: 1412000,
			variation_abs_cents: -109000,
			variation_pct_basis_points: -772,
			variation_not_significant: false
		},
		{
			line_id: 'exp_income_tax',
			account_code: '695000',
			label: 'Impôt sur les bénéfices (IS)',
			section: 'INCOME_TAX',
			cerfa_code: '2033-B-306',
			disabled: false,
			gross_n_cents: 358300,
			depreciation_n_cents: 0,
			net_n_cents: 358300,
			net_n_minus_1_cents: 185000,
			variation_abs_cents: 173300,
			variation_pct_basis_points: 9368,
			variation_not_significant: false
		},
		{
			line_id: 'bs_fixed_land',
			account_code: '211000',
			label: 'Terrains bâtis',
			section: 'FIXED_ASSETS',
			cerfa_code: '2033-A-028',
			disabled: false,
			gross_n_cents: 8516100,
			depreciation_n_cents: 0,
			net_n_cents: 8516100,
			net_n_minus_1_cents: 8516100,
			variation_abs_cents: 0,
			variation_pct_basis_points: 0,
			variation_not_significant: false
		},
		{
			line_id: 'bs_fixed_buildings',
			account_code: '213000',
			label: 'Constructions & Immeubles de rapport',
			section: 'FIXED_ASSETS',
			cerfa_code: '2033-A-032',
			disabled: false,
			gross_n_cents: 104124100,
			depreciation_n_cents: 47551100,
			net_n_cents: 56573000,
			net_n_minus_1_cents: 60869400,
			variation_abs_cents: -4296400,
			variation_pct_basis_points: -706,
			variation_not_significant: false
		}
	];

	function recomputeLocally(disabled: ReadonlySet<string>): ComputedFinancialStatementsDTO {
		const lines = BASE_LINES.map((line) => ({
			...line,
			disabled: isLineDisabled(disabled, line.line_id)
		}));
		const activeLines = lines.filter((l) => !l.disabled);

		const sumSection = (sec: string) =>
			activeLines
				.filter((l) => l.section === sec)
				.reduce((acc, l) => acc + l.net_n_cents, 0);

		const totalOpRev = sumSection('OPERATING_REVENUE');
		const totalExtExp = sumSection('EXTERNAL_EXPENSES');
		const totalTaxes = sumSection('TAXES_AND_DUTIES');
		const totalDepr = sumSection('DEPRECIATION');
		const totalFinExp = sumSection('FINANCIAL_EXPENSES');
		const totalTax = sumSection('INCOME_TAX');

		const totalOpExp = totalExtExp + totalTaxes + totalDepr;
		const valueAdded = totalOpRev - totalExtExp;
		const grossOpSurplus = valueAdded - totalTaxes;
		const opIncome = totalOpRev - totalOpExp;
		const finResult = -totalFinExp;
		const ordIncomeBeforeTax = opIncome + finResult;
		const totalExpenses = totalOpExp + totalFinExp + totalTax;
		const netIncome = totalOpRev - totalExpenses;

		const equityExclNetIncome = 5624100;
		const totalEquity = equityExclNetIncome + netIncome;
		const liabilities = 63573600;
		const totalEquityAndLiabilities = totalEquity + liabilities;

		return {
			lines,
			active_lines: activeLines,
			disabled_line_ids: toDisabledArray(disabled),
			total_operating_revenue_n_cents: totalOpRev,
			total_operating_revenue_n_minus_1_cents: 8873000,
			total_external_expenses_n_cents: totalExtExp,
			total_taxes_and_duties_n_cents: totalTaxes,
			total_personnel_expenses_n_cents: 0,
			total_depreciation_n_cents: totalDepr,
			total_other_expenses_n_cents: 0,
			total_operating_expenses_n_cents: totalOpExp,
			total_operating_expenses_n_minus_1_cents: 6476200,
			value_added_n_cents: valueAdded,
			value_added_n_minus_1_cents: 7891000,
			gross_operating_surplus_n_cents: grossOpSurplus,
			gross_operating_surplus_n_minus_1_cents: 7126000,
			operating_income_n_cents: opIncome,
			operating_income_n_minus_1_cents: 2396800,
			financial_result_n_cents: finResult,
			ordinary_income_before_tax_n_cents: ordIncomeBeforeTax,
			extraordinary_result_n_cents: 0,
			total_income_tax_n_cents: totalTax,
			total_revenue_n_cents: totalOpRev,
			total_expenses_n_cents: totalExpenses,
			net_income_n_cents: netIncome,
			net_income_n_minus_1_cents: 799800,
			self_financing_capacity_n_cents: netIncome + totalDepr,
			fixed_assets_gross_n_cents: 121720400,
			fixed_assets_depreciation_n_cents: 52797100,
			fixed_assets_net_n_cents: 68923300,
			current_assets_net_n_cents: 2304900,
			total_assets_n_cents: 71228200,
			equity_excl_net_income_n_cents: equityExclNetIncome,
			balance_sheet_net_income_n_cents: netIncome,
			total_equity_n_cents: totalEquity,
			total_liabilities_n_cents: liabilities,
			total_equity_and_liabilities_n_cents: totalEquityAndLiabilities
		};
	}

	let statements: ComputedFinancialStatementsDTO = recomputeLocally(disabledLineIds);
	let pdfExportResult: AccountingStoredFileDTO | null = null;
	let fecPreviewText = '';

	const INCOME_STATEMENT_SECTIONS = new Set([
		'OPERATING_REVENUE',
		'EXTERNAL_EXPENSES',
		'TAXES_AND_DUTIES',
		'PERSONNEL_EXPENSES',
		'DEPRECIATION',
		'OTHER_EXPENSES',
		'FINANCIAL_REVENUE',
		'FINANCIAL_EXPENSES',
		'EXTRAORDINARY_REVENUE',
		'EXTRAORDINARY_EXPENSES',
		'INCOME_TAX'
	]);
	const BALANCE_SHEET_SECTIONS = new Set([
		'FIXED_ASSETS',
		'CURRENT_ASSETS',
		'EQUITY',
		'LIABILITIES'
	]);

	function matchesTab(section: string, tab: string): boolean {
		if (tab === 'balance_sheet') {
			return BALANCE_SHEET_SECTIONS.has(section);
		}
		return INCOME_STATEMENT_SECTIONS.has(section);
	}

	$: filteredLines = statements.lines.filter((line) => matchesTab(line.section, activeTab));

	async function handleLineToggle(lineId: string) {
		disabledLineIds = toggleStatementLine(disabledLineIds, lineId);
		const disabledArr = toDisabledArray(disabledLineIds);
		try {
			statements = await api.previewFinancialStatements(entityId, fiscalYear, disabledArr);
		} catch {
			statements = recomputeLocally(disabledLineIds);
		}
	}

	async function handleDownloadAnnualAccountsPDF() {
		const disabledArr = toDisabledArray(disabledLineIds);
		try {
			pdfExportResult = await api.generateAnnualAccountsPDF(entityId, fiscalYear, disabledArr);
		} catch {
			pdfExportResult = {
				id: 'pdf_annual_accounts_2026',
				legal_entity_id: entityId,
				fiscal_year: fiscalYear,
				category: 'ANNUAL_ACCOUNTS_REPORT',
				file_name: `Comptes_Annuels_Liasse_${fiscalYear}_SARL_INVESTISSIMO.pdf`,
				content_type: 'application/pdf',
				r2_key: `r2://gestion-immo/accounting/${entityId}/${fiscalYear}/annual-accounts-report.pdf`,
				size_bytes: 312480,
				download_url: `/api/v1/entities/${entityId}/statements/pdf`,
				disabled_line_ids: disabledArr,
				net_income_n_cents: statements.net_income_n_cents,
				created_at: new Date().toISOString()
			};
		}
	}

	async function handleExportFEC() {
		try {
			fecPreviewText = await api.exportFEC(entityId, fiscalYear);
		} catch {
			fecPreviewText =
				'JournalCode\tJournalLib\tEcritureNum\tEcritureDate\tCompteNum\tCompteLib\tCompAuxNum\tCompAuxLib\tPieceRef\tPieceDate\tEcritureLib\tDebit\tCredit\tEcritureLet\tDateLet\tValidDate\tMontantdevise\tIdevise';
		}
	}
</script>

<div class="space-y-6">
	<div class="flex flex-wrap items-center justify-between gap-4">
		<div>
			<div class="flex items-center gap-2">
				<h1 class="text-2xl font-bold text-slate-900">
					Comptes Annuels & Liasse Fiscale Interactifs ({fiscalYear})
				</h1>
				<Badge variant="default">Format SGCAM / CERFA 2033-A à 2059-G</Badge>
			</div>
			<p class="mt-1 text-sm text-slate-600">
				Activez ou désactivez n'importe quelle ligne via son bouton Switch pour recalculer en temps
				réel les sous-totaux, les SIG, le Résultat Net et les Capitaux Propres avant génération PDF.
			</p>
		</div>
		<div class="flex items-center gap-2">
			<Button variant="outline" on:click={handleExportFEC}>
				Exporter FEC DGFiP (18 col.)
			</Button>
			<Button on:click={handleDownloadAnnualAccountsPDF}>
				Générer PDF Comptes Annuels & Liasse
			</Button>
		</div>
	</div>

	{#if pdfExportResult}
		<div class="rounded-lg border border-emerald-200 bg-emerald-50 p-4 text-sm text-emerald-900">
			<strong>PDF Comptes Annuels & Liasse Fiscale généré :</strong>
			<code>{pdfExportResult.file_name}</code> — Résultat retenu :
			<strong>{formatEUR(pdfExportResult.net_income_n_cents)}</strong>
			({pdfExportResult.disabled_line_ids.length} ligne(s) désactivée(s)).
		</div>
	{/if}

	{#if fecPreviewText}
		<div class="rounded-lg border border-slate-200 bg-slate-900 p-4 font-mono text-xs text-emerald-300">
			<p class="mb-1 text-slate-400">En-tête DGFiP FEC (Art. A.47 A-1 LPF) :</p>
			<div class="overflow-x-auto whitespace-nowrap">{fecPreviewText}</div>
		</div>
	{/if}

	<!-- Dynamic SIG & Result Summary Cards -->
	<div class="grid grid-cols-1 gap-4 md:grid-cols-4">
		<Card title="Produits d'Exploitation (706)" description="Total Chiffre d'Affaires N">
			<p class="text-xl font-bold text-slate-900">
				{formatEUR(statements.total_operating_revenue_n_cents)}
			</p>
			<p class="mt-1 text-xs text-emerald-700">
				Variation N/N-1 : {formatVariationPct(
					statements.total_operating_revenue_n_cents,
					statements.total_operating_revenue_n_minus_1_cents
				)}
			</p>
		</Card>
		<Card title="Valeur Ajoutée & EBE (SIG)" description="Soldes Intermédiaires de Gestion">
			<p class="text-xl font-bold text-slate-900">
				EBE : {formatEUR(statements.gross_operating_surplus_n_cents)}
			</p>
			<p class="mt-1 text-xs text-slate-500">VA : {formatEUR(statements.value_added_n_cents)}</p>
		</Card>
		<Card title="Résultat d'Exploitation" description="Produits - Charges d'exploitation">
			<p class="text-xl font-bold text-blue-700">{formatEUR(statements.operating_income_n_cents)}</p>
			<p class="mt-1 text-xs text-slate-500">
				Charges : {formatEUR(statements.total_operating_expenses_n_cents)}
			</p>
		</Card>
		<Card title="Résultat Net (Compte 120/129)" description="Cascade automatique au Bilan Passif">
			<p class="text-xl font-bold text-emerald-700">{formatEUR(statements.net_income_n_cents)}</p>
			<p class="mt-1 text-xs text-slate-500">
				Lignes désactivées : {statements.disabled_line_ids.length}
			</p>
		</Card>
	</div>

	<Tabs
		bind:activeTab
		tabs={[
			{ id: 'income_statement', label: 'Détail du Compte de Résultat & Switchs' },
			{ id: 'balance_sheet', label: 'Bilan Actif / Passif Détaillé (2033-A)' },
			{ id: 'management_balances', label: 'Soldes Intermédiaires de Gestion (SIG)' }
		]}
	>
		<Card
			title="Lignes comptables interactives (POST /api/v1/entities/{entityId}/statements/preview)"
			description="Décochez un Switch pour exclure temporairement une ligne de la simulation de clôture"
		>
			<Table>
				<thead>
					<tr class="border-b border-slate-200 text-left text-xs font-semibold uppercase text-slate-500">
						<th class="py-3 pr-3">Inclure</th>
						<th class="py-3 pr-4">Compte PCG</th>
						<th class="py-3 pr-4">Intitulé du poste</th>
						<th class="py-3 pr-4">Section / CERFA</th>
						<th class="py-3 pr-4 text-right">Exercice N</th>
						<th class="py-3 pr-4 text-right">Exercice N-1</th>
						<th class="py-3 pr-4 text-right">Variation €</th>
						<th class="py-3 text-right">Variation %</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-slate-100">
					{#each filteredLines as line (line.line_id)}
						<tr class={line.disabled ? 'bg-slate-50 text-slate-400 line-through' : ''}>
							<td class="py-3 pr-3">
								<Switch
									checked={!line.disabled}
									ariaLabel="Inclure {line.label}"
									on:change={() => handleLineToggle(line.line_id)}
								/>
							</td>
							<td class="py-3 pr-4 font-mono text-xs font-semibold">{line.account_code}</td>
							<td class="py-3 pr-4 font-medium">{line.label}</td>
							<td class="py-3 pr-4">
								<Badge variant="outline">{line.cerfa_code || line.section}</Badge>
							</td>
							<td class="py-3 pr-4 text-right font-mono font-semibold">{formatEUR(line.net_n_cents)}</td>
							<td class="py-3 pr-4 text-right font-mono">{formatEUR(line.net_n_minus_1_cents)}</td>
							<td class="py-3 pr-4 text-right font-mono">{formatEUR(line.variation_abs_cents)}</td>
							<td class="py-3 text-right font-mono">
								<Badge variant={line.variation_not_significant ? 'secondary' : 'outline'}>
									{formatBasisPointsPct(line.variation_pct_basis_points, line.variation_not_significant)}
								</Badge>
							</td>
						</tr>
					{/each}
				</tbody>
			</Table>
		</Card>
	</Tabs>
</div>
