<script lang="ts">
	import { resolve } from '$app/paths';
	import IconBook from '@tabler/icons-svelte/icons/book';
	import IconCalculator from '@tabler/icons-svelte/icons/calculator';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { Checkbox } from '#lib/components/ui/checkbox/index.js';
	import * as Empty from '#lib/components/ui/empty/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import * as Table from '#lib/components/ui/table/index.js';
	import { formatBasisPointsToPercent, formatCentsToEuro } from '#lib/core/formatters/money';
	import { formatStatementSectionLabel } from '#lib/modules/accounting/statement-state';
	import type { ComputedFinancialStatementsDTO } from '#lib/modules/accounting/types';

	let {
		statements,
		fiscalYear,
		disabledLineIds,
		isRecomputing,
		onToggleLine
	}: {
		statements: ComputedFinancialStatementsDTO | null;
		fiscalYear: number;
		disabledLineIds: string[];
		isRecomputing: boolean;
		onToggleLine: (lineId: string) => void;
	} = $props();
</script>

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
						Sélectionnez une entité juridique pour visualiser son bilan, son compte de résultat et
						ses soldes intermédiaires de gestion.
					</Empty.Description>
				</Empty.Header>
			</Empty.Root>
		</Card.Content>
	</Card.Root>
{:else}
	<div class="flex flex-col gap-6">
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
			<Card.Root>
				<Card.Header class="pb-2">
					<Card.Description>Produits d’exploitation (N)</Card.Description>
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
						Capacité d’Autofinancement (CAF) :
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
						Postes du Bilan & Compte de Résultat — Exercice {fiscalYear}
					</Card.Title>
					<Card.Description>
						Décochez une ligne (ex. Dotations aux amortissements 6811) pour recalculer
						instantanément le résultat net et la CAF.
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
					<div class="py-10">
						<Empty.Root>
							<Empty.Header>
								<Empty.Media variant="icon">
									<IconBook class="size-5" />
								</Empty.Media>
								<Empty.Title>Aucun poste de balance sur l’exercice {fiscalYear}</Empty.Title>
								<Empty.Description>
									Alimentez la balance PCG ou saisissez une écriture comptable en partie double pour
									calculer les états financiers.
								</Empty.Description>
							</Empty.Header>
							<Empty.Content>
								<Button
									size="sm"
									variant="outline"
									href={`${resolve('/(app)/accounting/journal')}?year=${fiscalYear}`}
								>
									<IconBook data-icon="inline-start" />
									Ouvrir le Journal & la Balance PCG
								</Button>
							</Empty.Content>
						</Empty.Root>
					</div>
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
											onCheckedChange={() => onToggleLine(line.line_id)}
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
	</div>
{/if}
