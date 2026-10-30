<script lang="ts">
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Table from '#lib/components/ui/table/index.js';
	import { formatCentsToEuro } from '#lib/core/formatters/money';
	import { formatEntityTypeLabel } from '#lib/modules/portfolio/labels';
	import type { EntityPortfolioSummaryDTO } from '#lib/modules/portfolio/types';

	let {
		summaries,
		selectedEntityId,
		onSelectEntity
	}: {
		summaries: EntityPortfolioSummaryDTO[];
		selectedEntityId: string;
		onSelectEntity: (entityId: string) => void;
	} = $props();
</script>

<Card.Root class="lg:col-span-7">
	<Card.Header class="border-b pb-3">
		<Card.Title class="text-sm font-semibold">Structures juridiques</Card.Title>
		<Card.Description>
			Cliquez sur une entité pour afficher la composition de son capital (CERFA 2059-F)
		</Card.Description>
	</Card.Header>
	<Card.Content class="p-0">
		<Table.Root>
			<Table.Header>
				<Table.Row>
					<Table.Head class="pl-4">Raison sociale</Table.Head>
					<Table.Head>Forme</Table.Head>
					<Table.Head class="text-right">Capital</Table.Head>
					<Table.Head class="text-right">VNC Patrimoine</Table.Head>
					<Table.Head class="pr-4 text-right">Associés</Table.Head>
				</Table.Row>
			</Table.Header>
			<Table.Body>
				{#each summaries as summary (summary.entity.id)}
					<Table.Row class="cursor-pointer" onclick={() => onSelectEntity(summary.entity.id)}>
						<Table.Cell class="pl-4">
							<div class="flex flex-col">
								<span class="font-medium">{summary.entity.name}</span>
								<span class="font-mono text-[11px] text-muted-foreground">
									SIRET {summary.entity.siret || '—'} · NAF {summary.entity.naf_code || '6820B'}
								</span>
							</div>
						</Table.Cell>
						<Table.Cell>
							<Badge variant="outline">
								{formatEntityTypeLabel(summary.entity.entity_type)}
							</Badge>
						</Table.Cell>
						<Table.Cell class="text-right font-mono tabular-nums">
							{formatCentsToEuro(summary.entity.share_capital_cents)}
							<span class="block text-[10px] text-muted-foreground">
								{summary.entity.total_shares} parts
							</span>
						</Table.Cell>
						<Table.Cell class="text-right font-mono tabular-nums">
							{formatCentsToEuro(summary.total_book_value_cents)}
						</Table.Cell>
						<Table.Cell class="pr-4 text-right">
							<Button
								type="button"
								size="sm"
								variant={selectedEntityId === summary.entity.id ? 'default' : 'outline'}
								onclick={(event: MouseEvent) => {
									event.stopPropagation();
									onSelectEntity(summary.entity.id);
								}}
							>
								{summary.shareholders.length} associé(s)
							</Button>
						</Table.Cell>
					</Table.Row>
				{/each}
			</Table.Body>
		</Table.Root>
	</Card.Content>
</Card.Root>
