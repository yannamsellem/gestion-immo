<script lang="ts">
	import IconPlus from '@tabler/icons-svelte/icons/plus';
	import IconTool from '@tabler/icons-svelte/icons/tool';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Empty from '#lib/components/ui/empty/index.js';
	import * as Table from '#lib/components/ui/table/index.js';
	import { formatIsoDateFr } from '#lib/core/formatters/date';
	import { formatCentsToEuro } from '#lib/core/formatters/money';
	import {
		formatIncidentStatusLabel,
		formatIncidentStatusVariant
	} from '#lib/modules/inspection/labels';
	import type { IncidentDTO } from '#lib/modules/inspection/types';
	import type { LeaseDTO, TenantDTO } from '#lib/modules/lease/types';
	import type { PropertyDTO } from '#lib/modules/portfolio/types';

	let {
		incidents,
		properties,
		leases,
		tenants,
		onCreateIncident,
		onEditIncident
	}: {
		incidents: IncidentDTO[];
		properties: PropertyDTO[];
		leases: LeaseDTO[];
		tenants: TenantDTO[];
		onCreateIncident: () => void;
		onEditIncident: (incident: IncidentDTO) => void;
	} = $props();

	const propertyMap = $derived(new Map(properties.map((property) => [property.id, property])));
	const tenantMap = $derived(new Map(tenants.map((tenant) => [tenant.id, tenant])));
	const leaseMap = $derived(new Map(leases.map((lease) => [lease.id, lease])));

	function formatLeaseDisplayLabel(lease: LeaseDTO): string {
		const tenant = tenantMap.get(lease.tenant_id);
		return `${tenant?.full_name ?? 'Locataire'} (Bail ${lease.id.slice(0, 6)})`;
	}
</script>

{#if incidents.length === 0}
	<Card.Root>
		<Card.Content class="py-10">
			<Empty.Root>
				<Empty.Header>
					<Empty.Media variant="icon">
						<IconTool class="size-5" />
					</Empty.Media>
					<Empty.Title>Aucun incident ou sinistre déclaré</Empty.Title>
					<Empty.Description>
						Enregistrez les dégâts des eaux, pannes ou travaux pour assurer le suivi assurance et le
						chiffrage des réparations.
					</Empty.Description>
				</Empty.Header>
				<Empty.Content>
					<Button onclick={onCreateIncident}>
						<IconPlus data-icon="inline-start" />
						Déclarer un incident
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
						<Table.Head class="pl-4">Date</Table.Head>
						<Table.Head>Intitulé de l’incident</Table.Head>
						<Table.Head>Bien / Bail rattaché</Table.Head>
						<Table.Head>Réf. Sinistre</Table.Head>
						<Table.Head class="text-right">Coût estimé</Table.Head>
						<Table.Head>Statut</Table.Head>
						<Table.Head class="pr-4 text-right">Action</Table.Head>
					</Table.Row>
				</Table.Header>
				<Table.Body>
					{#each incidents as incident (incident.id)}
						<Table.Row class="cursor-pointer" onclick={() => onEditIncident(incident)}>
							<Table.Cell class="pl-4 text-xs">
								{formatIsoDateFr(incident.occurred_at)}
							</Table.Cell>
							<Table.Cell class="font-medium">{incident.title}</Table.Cell>
							<Table.Cell class="text-xs text-muted-foreground">
								{#if incident.property_id && propertyMap.get(incident.property_id)}
									{propertyMap.get(incident.property_id)?.label}
								{:else if incident.lease_id && leaseMap.get(incident.lease_id)}
									{formatLeaseDisplayLabel(leaseMap.get(incident.lease_id)!)}
								{:else}
									—
								{/if}
							</Table.Cell>
							<Table.Cell class="font-mono text-xs">
								{incident.insurance_claim_number || '—'}
							</Table.Cell>
							<Table.Cell class="text-right font-mono tabular-nums">
								{formatCentsToEuro(incident.estimated_cost_cents)}
							</Table.Cell>
							<Table.Cell>
								<Badge variant={formatIncidentStatusVariant(incident.status)}>
									{formatIncidentStatusLabel(incident.status)}
								</Badge>
							</Table.Cell>
							<Table.Cell class="pr-4 text-right">
								<Button
									size="sm"
									variant="outline"
									onclick={(event: MouseEvent) => {
										event.stopPropagation();
										onEditIncident(incident);
									}}
								>
									Traiter →
								</Button>
							</Table.Cell>
						</Table.Row>
					{/each}
				</Table.Body>
			</Table.Root>
		</Card.Content>
	</Card.Root>
{/if}
