<script lang="ts">
	import { resolve } from '$app/paths';
	import IconCheck from '@tabler/icons-svelte/icons/check';
	import IconPlus from '@tabler/icons-svelte/icons/plus';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Table from '#lib/components/ui/table/index.js';
	import { formatCentsToEuro } from '#lib/core/formatters/money';
	import type { EntityCockpitGroup } from '#lib/modules/cockpit/types';
	import { formatEntityTypeLabel, formatLotTypeLabel } from '#lib/modules/portfolio/labels';

	let {
		group
	}: {
		group: EntityCockpitGroup;
	} = $props();
</script>

<Card.Root>
	<Card.Header
		class="flex flex-col justify-between gap-3 border-b pb-3 sm:flex-row sm:items-center"
	>
		<div class="flex flex-wrap items-center gap-2">
			<Card.Title class="text-sm font-semibold">{group.entity.name}</Card.Title>
			<Badge variant="outline">{formatEntityTypeLabel(group.entity.entity_type)}</Badge>
			{#if group.entity.siret}
				<span class="font-mono text-xs text-muted-foreground">SIRET {group.entity.siret}</span>
			{/if}
		</div>
		<div class="flex flex-wrap items-center gap-3">
			<span class="text-xs text-muted-foreground tabular-nums">
				Encaissé :
				<strong class="font-semibold text-foreground">
					{formatCentsToEuro(group.collectedRentCents)}
				</strong>
				/ {formatCentsToEuro(group.expectedRentCents)}
			</span>
			{#if group.generateCallsHref}
				<Button size="sm" variant="secondary" href={group.generateCallsHref}>
					Générer appels du mois ({group.uncalledLeasesCount}) →
				</Button>
			{/if}
		</div>
	</Card.Header>

	<Card.Content class="p-0">
		{#if group.rows.length === 0}
			<div class="flex items-center justify-between px-4 py-6">
				<p class="text-xs text-muted-foreground">Aucun bien immobilier rattaché à cette entité.</p>
				<Button
					size="sm"
					variant="outline"
					href={`${resolve('/(app)/properties')}?entityId=${encodeURIComponent(group.entity.id)}&action=create`}
				>
					<IconPlus data-icon="inline-start" />
					Ajouter un bien →
				</Button>
			</div>
		{:else}
			<Table.Root>
				<Table.Header>
					<Table.Row>
						<Table.Head class="pl-4">Bien & Lot</Table.Head>
						<Table.Head>Locataire / Statut</Table.Head>
						<Table.Head class="text-right">Loyer CC</Table.Head>
						<Table.Head class="text-right">Reste dû</Table.Head>
						<Table.Head class="pr-4 text-right">Actions à traiter</Table.Head>
					</Table.Row>
				</Table.Header>
				<Table.Body>
					{#each group.rows as row (`${row.propertyId}-${row.lotId}`)}
						<Table.Row>
							<Table.Cell class="pl-4">
								<div class="flex flex-col gap-0.5">
									<span class="font-medium">
										{row.propertyLabel}
										{#if row.lotLabel !== row.propertyLabel}
											— {row.lotLabel}
										{/if}
									</span>
									<span class="text-[11px] text-muted-foreground">
										{formatLotTypeLabel(row.lotType)}
									</span>
								</div>
							</Table.Cell>

							<Table.Cell>
								{#if row.isVacant}
									<Badge variant="outline">Lot vacant</Badge>
								{:else}
									<span class="font-medium">{row.tenantName}</span>
								{/if}
							</Table.Cell>

							<Table.Cell class="text-right font-mono tabular-nums">
								{#if row.isVacant}
									—
								{:else}
									{formatCentsToEuro(row.monthlyRentIncludingChargesCents)}
								{/if}
							</Table.Cell>

							<Table.Cell class="text-right font-mono tabular-nums">
								{#if row.isVacant}
									—
								{:else if row.remainingDueCents > 0}
									<span class="font-semibold text-destructive">
										{formatCentsToEuro(row.remainingDueCents)}
									</span>
								{:else}
									<span class="text-muted-foreground">0,00 €</span>
								{/if}
							</Table.Cell>

							<Table.Cell class="pr-4 text-right">
								<div class="flex flex-wrap items-center justify-end gap-1.5">
									{#if row.isUpToDate && row.actions.length === 0}
										<Badge variant="secondary">
											<IconCheck />
											À jour
										</Badge>
									{:else}
										{#each row.actions as action (`${row.lotId}-${action.kind}-${action.href}`)}
											<Button size="sm" variant={action.variant} href={action.href}>
												{action.label}
											</Button>
										{/each}
									{/if}
								</div>
							</Table.Cell>
						</Table.Row>
					{/each}
				</Table.Body>
			</Table.Root>
		{/if}
	</Card.Content>
</Card.Root>
