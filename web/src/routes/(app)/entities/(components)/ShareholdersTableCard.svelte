<script lang="ts">
	import IconPlus from '@tabler/icons-svelte/icons/plus';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Table from '#lib/components/ui/table/index.js';
	import { formatBasisPointsToPercent, formatCentsToEuro } from '#lib/core/formatters/money';
	import type { EntityPortfolioSummaryDTO } from '#lib/modules/portfolio/types';

	let {
		summary,
		onAddShareholder
	}: {
		summary: EntityPortfolioSummaryDTO | null;
		onAddShareholder: (entityId: string) => void;
	} = $props();
</script>

<Card.Root class="lg:col-span-5">
	<Card.Header class="flex flex-row items-center justify-between border-b pb-3">
		<div>
			<Card.Title class="text-sm font-semibold">
				Associés & Comptes Courants — {summary?.entity.name ?? ''}
			</Card.Title>
			<Card.Description>
				Répartition en points de base (10000 = 100,00 %) pour le feuillet CERFA 2059-F
			</Card.Description>
		</div>
		{#if summary}
			<Button
				type="button"
				size="sm"
				variant="secondary"
				onclick={() => onAddShareholder(summary.entity.id)}
			>
				<IconPlus data-icon="inline-start" />
				Associé
			</Button>
		{/if}
	</Card.Header>
	<Card.Content class="p-0">
		{#if !summary || summary.shareholders.length === 0}
			<p class="px-4 py-6 text-xs text-muted-foreground">
				Aucun associé renseigné pour cette structure.
			</p>
		{:else}
			<Table.Root>
				<Table.Header>
					<Table.Row>
						<Table.Head class="pl-4">Associé</Table.Head>
						<Table.Head class="text-right">Quote-part</Table.Head>
						<Table.Head class="pr-4 text-right">C/C Associé (455)</Table.Head>
					</Table.Row>
				</Table.Header>
				<Table.Body>
					{#each summary.shareholders as shareholder (shareholder.id)}
						<Table.Row>
							<Table.Cell class="pl-4">
								<div class="flex flex-col">
									<span class="font-medium">{shareholder.full_name}</span>
									<span class="text-[11px] text-muted-foreground">
										{shareholder.shares_count} parts
									</span>
								</div>
							</Table.Cell>
							<Table.Cell class="text-right font-mono tabular-nums">
								<Badge variant="secondary">
									{formatBasisPointsToPercent(shareholder.detention_basis_points)}
								</Badge>
							</Table.Cell>
							<Table.Cell class="pr-4 text-right font-mono tabular-nums">
								{formatCentsToEuro(shareholder.current_account_cents)}
							</Table.Cell>
						</Table.Row>
					{/each}
				</Table.Body>
			</Table.Root>
		{/if}
	</Card.Content>
</Card.Root>
