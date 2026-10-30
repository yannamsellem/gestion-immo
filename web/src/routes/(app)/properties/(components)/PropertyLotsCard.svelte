<script lang="ts">
	import { resolve } from '$app/paths';
	import IconPlus from '@tabler/icons-svelte/icons/plus';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Table from '#lib/components/ui/table/index.js';
	import { formatCentsToEuro } from '#lib/core/formatters/money';
	import type { LeaseDTO } from '#lib/modules/lease/types';
	import { formatAccountingCategoryLabel, formatLotTypeLabel } from '#lib/modules/portfolio/labels';
	import type { PropertyDTO } from '#lib/modules/portfolio/types';

	let {
		property,
		entityName,
		activeLeaseByLotId,
		onAddLot
	}: {
		property: PropertyDTO;
		entityName: string;
		activeLeaseByLotId: ReadonlyMap<string, LeaseDTO>;
		onAddLot: (propertyId: string) => void;
	} = $props();
</script>

<Card.Root>
	<Card.Header
		class="flex flex-col justify-between gap-2 border-b pb-3 sm:flex-row sm:items-center"
	>
		<div class="flex flex-col gap-1">
			<div class="flex flex-wrap items-center gap-2">
				<Card.Title class="text-sm font-semibold">{property.label}</Card.Title>
				<Badge variant="outline">{entityName}</Badge>
				<Badge variant={property.accounting_category === 'FIXED_ASSET' ? 'secondary' : 'default'}>
					{formatAccountingCategoryLabel(property.accounting_category)}
				</Badge>
			</div>
			<Card.Description>{property.address}</Card.Description>
		</div>

		<div class="flex flex-wrap items-center gap-3">
			<div class="text-right text-xs">
				<span class="text-muted-foreground">VNC :</span>
				<strong class="font-mono font-semibold">
					{formatCentsToEuro(property.total_book_value_cents)}
				</strong>
				<span class="block text-[11px] text-muted-foreground">
					Acq. {formatCentsToEuro(property.acquisition_cents)} · Travaux
					{formatCentsToEuro(property.work_stock_cents)}
				</span>
			</div>
			<Button type="button" size="sm" variant="secondary" onclick={() => onAddLot(property.id)}>
				<IconPlus data-icon="inline-start" />
				Ajouter un lot
			</Button>
		</div>
	</Card.Header>

	<Card.Content class="p-0">
		{#if !property.lots || property.lots.length === 0}
			<div class="flex items-center justify-between px-4 py-4">
				<p class="text-xs text-muted-foreground">Aucun lot découpé dans ce bien immobilier.</p>
				<Button type="button" size="sm" variant="outline" onclick={() => onAddLot(property.id)}>
					Créer le premier lot →
				</Button>
			</div>
		{:else}
			<Table.Root>
				<Table.Header>
					<Table.Row>
						<Table.Head class="pl-4">Désignation du lot</Table.Head>
						<Table.Head>Type</Table.Head>
						<Table.Head>Étage</Table.Head>
						<Table.Head class="text-right">Surface</Table.Head>
						<Table.Head class="text-right">Tantièmes</Table.Head>
						<Table.Head>Occupation</Table.Head>
						<Table.Head class="pr-4 text-right">Action</Table.Head>
					</Table.Row>
				</Table.Header>
				<Table.Body>
					{#each property.lots as lot (lot.id)}
						{@const activeLease = activeLeaseByLotId.get(lot.id)}
						<Table.Row>
							<Table.Cell class="pl-4 font-medium">{lot.label}</Table.Cell>
							<Table.Cell class="text-xs">
								{formatLotTypeLabel(lot.lot_type)}
							</Table.Cell>
							<Table.Cell class="text-xs text-muted-foreground">
								{lot.floor || '—'}
							</Table.Cell>
							<Table.Cell class="text-right font-mono text-xs tabular-nums">
								{lot.surface_m2} m²
							</Table.Cell>
							<Table.Cell class="text-right font-mono text-xs tabular-nums">
								{lot.ownership_share} / 1000
							</Table.Cell>
							<Table.Cell>
								{#if activeLease}
									<Badge variant="secondary">Loué</Badge>
								{:else}
									<Badge variant="outline">Vacant</Badge>
								{/if}
							</Table.Cell>
							<Table.Cell class="pr-4 text-right">
								{#if activeLease}
									<Button
										size="sm"
										variant="outline"
										href={`${resolve('/(app)/rents')}?entityId=${encodeURIComponent(property.legal_entity_id)}&leaseId=${encodeURIComponent(activeLease.id)}`}
									>
										Voir loyers →
									</Button>
								{:else}
									<Button
										size="sm"
										variant="default"
										href={`${resolve('/(app)/leases')}?entityId=${encodeURIComponent(property.legal_entity_id)}&lotId=${encodeURIComponent(lot.id)}`}
									>
										Créer un bail →
									</Button>
								{/if}
							</Table.Cell>
						</Table.Row>
					{/each}
				</Table.Body>
			</Table.Root>
		{/if}
	</Card.Content>
</Card.Root>
