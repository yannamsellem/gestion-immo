<script lang="ts">
	import IconCalculator from '@tabler/icons-svelte/icons/calculator';
	import IconPlus from '@tabler/icons-svelte/icons/plus';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Empty from '#lib/components/ui/empty/index.js';
	import * as Table from '#lib/components/ui/table/index.js';
	import { formatBasisPointsToPercent, formatCentsToEuro } from '#lib/core/formatters/money';
	import { formatStatementSectionLabel } from '#lib/modules/accounting/statement-state';
	import type { ComputedLineDTO } from '#lib/modules/accounting/types';

	let {
		lines,
		fiscalYear,
		onCreateLine
	}: {
		lines: ComputedLineDTO[];
		fiscalYear: number;
		onCreateLine: () => void;
	} = $props();
</script>

<Card.Root>
	<Card.Header
		class="flex flex-col justify-between gap-2 border-b pb-3 sm:flex-row sm:items-center"
	>
		<div>
			<Card.Title class="text-sm font-semibold">
				Balance générale PCG — Exercice {fiscalYear} ({lines.length} postes)
			</Card.Title>
			<Card.Description>
				Soldes nets N et N-1 alimentant le bilan, le compte de résultat et les cases CERFA
				2072/2033.
			</Card.Description>
		</div>
		<Button size="sm" variant="outline" onclick={onCreateLine}>
			<IconPlus data-icon="inline-start" />
			Ligne de balance
		</Button>
	</Card.Header>

	<Card.Content class="p-0">
		{#if lines.length === 0}
			<div class="py-10">
				<Empty.Root>
					<Empty.Header>
						<Empty.Media variant="icon">
							<IconCalculator class="size-5" />
						</Empty.Media>
						<Empty.Title>Aucun poste de balance enregistré</Empty.Title>
						<Empty.Description>
							Ajoutez un compte PCG (ex. 706000 Loyers, 681100 Amortissements, 213000 Immeubles)
							pour initialiser la balance de l’exercice {fiscalYear}.
						</Empty.Description>
					</Empty.Header>
					<Empty.Content>
						<Button size="sm" variant="outline" onclick={onCreateLine}>
							<IconPlus data-icon="inline-start" />
							Ajouter une ligne de balance
						</Button>
					</Empty.Content>
				</Empty.Root>
			</div>
		{:else}
			<Table.Root>
				<Table.Header>
					<Table.Row>
						<Table.Head class="pl-4">Compte PCG</Table.Head>
						<Table.Head>Intitulé du poste</Table.Head>
						<Table.Head>Rubrique & CERFA</Table.Head>
						<Table.Head class="text-right">Net {fiscalYear}</Table.Head>
						<Table.Head class="text-right">Net {fiscalYear - 1}</Table.Head>
						<Table.Head class="pr-4 text-right">Variation</Table.Head>
					</Table.Row>
				</Table.Header>
				<Table.Body>
					{#each lines as line (line.line_id)}
						<Table.Row>
							<Table.Cell class="pl-4 font-mono text-xs font-semibold">
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
