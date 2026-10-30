<script lang="ts" module>
	import type { LeaseDTO, RentCallDTO } from '#lib/modules/lease/types';
	import type { NegativePositionDTO } from '#lib/modules/payment/types';
	import type { LegalEntityDTO } from '#lib/modules/portfolio/types';

	export type RentStatusFilter = 'ALL' | 'TO_COLLECT' | 'OVERDUE' | 'PAID';

	export interface UnifiedRentRow {
		lease: LeaseDTO;
		entity: LegalEntityDTO;
		propertyAndLotLabel: string;
		tenantName: string;
		rentCall: RentCallDTO | null;
		negativePosition: NegativePositionDTO | null;
		expectedGrossCents: number;
		remainingDueCents: number;
		statusCategory: 'PAID' | 'OVERDUE' | 'TO_COLLECT';
		statusBadgeLabel: string;
		statusBadgeVariant: 'default' | 'secondary' | 'destructive' | 'outline';
	}
</script>

<script lang="ts">
	import { resolve } from '$app/paths';
	import IconReceiptEuro from '@tabler/icons-svelte/icons/receipt-euro';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Empty from '#lib/components/ui/empty/index.js';
	import * as Table from '#lib/components/ui/table/index.js';
	import * as ToggleGroup from '#lib/components/ui/toggle-group/index.js';
	import { formatCentsToEuro } from '#lib/core/formatters/money';

	let {
		rows,
		statusFilter = $bindable<RentStatusFilter>('ALL'),
		onOpenRow
	}: {
		rows: UnifiedRentRow[];
		statusFilter?: RentStatusFilter;
		onOpenRow: (row: UnifiedRentRow, action?: 'pay' | 'remind' | null) => void;
	} = $props();

	const filteredRows = $derived(
		rows.filter((row) => statusFilter === 'ALL' || row.statusCategory === statusFilter)
	);
</script>

<div class="flex flex-wrap items-center justify-between gap-3">
	<ToggleGroup.Root
		type="single"
		variant="outline"
		size="sm"
		value={statusFilter}
		onValueChange={(next) => {
			if (next) statusFilter = next as RentStatusFilter;
		}}
	>
		<ToggleGroup.Item value="ALL">Tous ({rows.length})</ToggleGroup.Item>
		<ToggleGroup.Item value="TO_COLLECT">
			À encaisser ({rows.filter((row) => row.statusCategory === 'TO_COLLECT').length})
		</ToggleGroup.Item>
		<ToggleGroup.Item value="OVERDUE">
			Impayés ({rows.filter((row) => row.statusCategory === 'OVERDUE').length})
		</ToggleGroup.Item>
		<ToggleGroup.Item value="PAID">
			Payés ({rows.filter((row) => row.statusCategory === 'PAID').length})
		</ToggleGroup.Item>
	</ToggleGroup.Root>
</div>

{#if filteredRows.length === 0}
	<Card.Root>
		<Card.Content class="py-10">
			<Empty.Root>
				<Empty.Header>
					<Empty.Media variant="icon">
						<IconReceiptEuro class="size-5" />
					</Empty.Media>
					<Empty.Title>Aucun bail actif pour ce filtre</Empty.Title>
					<Empty.Description>
						Créez un bail dans la section Baux & Locataires ou modifiez le filtre de statut.
					</Empty.Description>
				</Empty.Header>
				<Empty.Content>
					<Button href={`${resolve('/(app)/leases')}?action=create`} variant="outline">
						Aller aux baux →
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
						<Table.Head class="pl-4">Entité & Lot</Table.Head>
						<Table.Head>Locataire</Table.Head>
						<Table.Head class="text-right">Appelé TTC / CC</Table.Head>
						<Table.Head class="text-right">Reste à payer</Table.Head>
						<Table.Head>Statut</Table.Head>
						<Table.Head class="pr-4 text-right">Actions</Table.Head>
					</Table.Row>
				</Table.Header>
				<Table.Body>
					{#each filteredRows as row (row.lease.id)}
						<Table.Row class="cursor-pointer" onclick={() => onOpenRow(row, null)}>
							<Table.Cell class="pl-4">
								<div class="flex flex-col gap-0.5">
									<span class="font-medium">{row.propertyAndLotLabel}</span>
									<span class="text-[11px] text-muted-foreground">{row.entity.name}</span>
								</div>
							</Table.Cell>

							<Table.Cell>
								<span class="font-medium">{row.tenantName}</span>
							</Table.Cell>

							<Table.Cell class="text-right font-mono tabular-nums">
								{formatCentsToEuro(row.expectedGrossCents)}
							</Table.Cell>

							<Table.Cell class="text-right font-mono tabular-nums">
								{#if row.remainingDueCents > 0}
									<span class="font-semibold text-destructive">
										{formatCentsToEuro(row.remainingDueCents)}
									</span>
								{:else}
									<span class="text-muted-foreground">0,00 €</span>
								{/if}
							</Table.Cell>

							<Table.Cell>
								<Badge variant={row.statusBadgeVariant}>{row.statusBadgeLabel}</Badge>
							</Table.Cell>

							<Table.Cell class="pr-4 text-right">
								<div class="flex items-center justify-end gap-1.5">
									{#if row.negativePosition && row.negativePosition.total_unpaid_cents > 0}
										<Button
											size="sm"
											variant="destructive"
											onclick={(event: MouseEvent) => {
												event.stopPropagation();
												onOpenRow(row, 'remind');
											}}
										>
											Relancer / GLI →
										</Button>
									{/if}
									<Button
										size="sm"
										variant={row.remainingDueCents > 0 ? 'default' : 'outline'}
										onclick={(event: MouseEvent) => {
											event.stopPropagation();
											onOpenRow(row, 'pay');
										}}
									>
										{row.remainingDueCents > 0 ? 'Saisir loyer →' : 'Quittance / Détail →'}
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
