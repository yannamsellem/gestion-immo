<script lang="ts">
	import { resolve } from '$app/paths';
	import IconFileDescription from '@tabler/icons-svelte/icons/file-description';
	import IconPlus from '@tabler/icons-svelte/icons/plus';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Empty from '#lib/components/ui/empty/index.js';
	import * as Table from '#lib/components/ui/table/index.js';
	import { formatBasisPointsToPercent, formatCentsToEuro } from '#lib/core/formatters/money';
	import { formatLeaseTypeLabel } from '#lib/modules/lease/labels';
	import type { LeaseDTO, TenantDTO } from '#lib/modules/lease/types';
	import type { LegalEntityDTO, LotDTO, PropertyDTO } from '#lib/modules/portfolio/types';

	let {
		leases,
		entities,
		properties,
		tenants,
		onCreateLease,
		onOpenDocuments
	}: {
		leases: LeaseDTO[];
		entities: LegalEntityDTO[];
		properties: PropertyDTO[];
		tenants: TenantDTO[];
		onCreateLease: () => void;
		onOpenDocuments: (lease: LeaseDTO) => void;
	} = $props();

	const entityMap = $derived(new Map(entities.map((entity) => [entity.id, entity])));
	const tenantMap = $derived(new Map(tenants.map((tenant) => [tenant.id, tenant])));
	const lotInfoMap = $derived(
		new Map<string, { lot: LotDTO; property: PropertyDTO }>(
			properties.flatMap((property) =>
				(property.lots ?? []).map((lot) => [lot.id, { lot, property }] as const)
			)
		)
	);
</script>

{#if leases.length === 0}
	<Card.Root>
		<Card.Content class="py-10">
			<Empty.Root>
				<Empty.Header>
					<Empty.Media variant="icon">
						<IconFileDescription class="size-5" />
					</Empty.Media>
					<Empty.Title>Aucun contrat de bail enregistré</Empty.Title>
					<Empty.Description>
						Créez un locataire puis associez-le à un lot immobilier pour activer la génération des
						appels de loyers.
					</Empty.Description>
				</Empty.Header>
				<Empty.Content>
					<Button onclick={onCreateLease}>
						<IconPlus data-icon="inline-start" />
						Créer un bail locatif
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
						<Table.Head class="pl-4">Bien & Lot</Table.Head>
						<Table.Head>Locataire</Table.Head>
						<Table.Head>Régime du bail</Table.Head>
						<Table.Head class="text-right">Loyer HC + Charges</Table.Head>
						<Table.Head>TVA / GLI</Table.Head>
						<Table.Head class="pr-4 text-right">Documents & Loyers</Table.Head>
					</Table.Row>
				</Table.Header>
				<Table.Body>
					{#each leases as lease (lease.id)}
						{@const lotInfo = lotInfoMap.get(lease.lot_id)}
						{@const tenant = tenantMap.get(lease.tenant_id)}
						<Table.Row>
							<Table.Cell class="pl-4">
								<div class="flex flex-col">
									<span class="font-medium">
										{#if lotInfo}
											{lotInfo.property.label} — {lotInfo.lot.label}
										{:else}
											Lot {lease.lot_id.slice(0, 8)}
										{/if}
									</span>
									<span class="text-[11px] text-muted-foreground">
										{entityMap.get(lease.legal_entity_id)?.name ?? ''} · Échéance le
										{lease.due_day_of_month} du mois
									</span>
								</div>
							</Table.Cell>
							<Table.Cell class="font-medium">
								{tenant?.full_name ?? 'Locataire'}
							</Table.Cell>
							<Table.Cell>
								<Badge variant="outline">{formatLeaseTypeLabel(lease.lease_type)}</Badge>
							</Table.Cell>
							<Table.Cell class="text-right font-mono tabular-nums">
								{formatCentsToEuro(lease.rent_net_cents)}
								<span class="block text-[11px] text-muted-foreground">
									+ {formatCentsToEuro(lease.charges_cents)} prov.
								</span>
							</Table.Cell>
							<Table.Cell class="text-xs">
								{#if lease.subject_to_vat}
									<Badge variant="secondary">
										TVA {formatBasisPointsToPercent(lease.vat_rate_basis_points)}
									</Badge>
								{/if}
								{#if lease.gli_policy_number}
									<span class="block font-mono text-[11px] text-muted-foreground">
										GLI {lease.gli_policy_number}
									</span>
								{/if}
							</Table.Cell>
							<Table.Cell class="pr-4 text-right">
								<div class="flex items-center justify-end gap-1.5">
									<Button size="sm" variant="outline" onclick={() => onOpenDocuments(lease)}>
										Pièces / Contrat →
									</Button>
									<Button
										size="sm"
										variant="secondary"
										href={`${resolve('/(app)/rents')}?entityId=${encodeURIComponent(lease.legal_entity_id)}&leaseId=${encodeURIComponent(lease.id)}`}
									>
										Loyers →
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
