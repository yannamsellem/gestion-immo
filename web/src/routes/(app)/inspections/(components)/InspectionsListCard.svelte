<script lang="ts">
	import IconClipboardCheck from '@tabler/icons-svelte/icons/clipboard-check';
	import IconDownload from '@tabler/icons-svelte/icons/download';
	import IconPlus from '@tabler/icons-svelte/icons/plus';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Empty from '#lib/components/ui/empty/index.js';
	import * as Table from '#lib/components/ui/table/index.js';
	import { formatIsoDateFr } from '#lib/core/formatters/date';
	import { formatInspectionTypeLabel } from '#lib/modules/inspection/labels';
	import type { InspectionDTO, InspectionTemplateDTO } from '#lib/modules/inspection/types';
	import type { LeaseDTO, TenantDTO } from '#lib/modules/lease/types';

	let {
		inspections,
		leases,
		tenants,
		templates,
		onCreateInspection,
		onOpenInspection
	}: {
		inspections: InspectionDTO[];
		leases: LeaseDTO[];
		tenants: TenantDTO[];
		templates: InspectionTemplateDTO[];
		onCreateInspection: () => void;
		onOpenInspection: (inspection: InspectionDTO) => void;
	} = $props();

	const tenantMap = $derived(new Map(tenants.map((tenant) => [tenant.id, tenant])));
	const leaseMap = $derived(new Map(leases.map((lease) => [lease.id, lease])));
	const templateMap = $derived(new Map(templates.map((tpl) => [tpl.id, tpl])));

	function formatLeaseDisplayLabel(lease: LeaseDTO): string {
		const tenant = tenantMap.get(lease.tenant_id);
		return `${tenant?.full_name ?? 'Locataire'} (Bail ${lease.id.slice(0, 6)})`;
	}
</script>

{#if inspections.length === 0}
	<Card.Root>
		<Card.Content class="py-10">
			<Empty.Root>
				<Empty.Header>
					<Empty.Media variant="icon">
						<IconClipboardCheck class="size-5" />
					</Empty.Media>
					<Empty.Title>Aucun état des lieux enregistré</Empty.Title>
					<Empty.Description>
						Démarrez un état des lieux contradictoire d’entrée ou de sortie à partir d’un modèle de
						grille pièce par pièce.
					</Empty.Description>
				</Empty.Header>
				<Empty.Content>
					<Button onclick={onCreateInspection}>
						<IconPlus data-icon="inline-start" />
						Nouvel état des lieux
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
						<Table.Head>Type</Table.Head>
						<Table.Head>Bail & Locataire</Table.Head>
						<Table.Head>Modèle de grille</Table.Head>
						<Table.Head>Clés & Éléments</Table.Head>
						<Table.Head class="pr-4 text-right">Rapport & Grille</Table.Head>
					</Table.Row>
				</Table.Header>
				<Table.Body>
					{#each inspections as inspection (inspection.id)}
						{@const lease = leaseMap.get(inspection.lease_id)}
						{@const template = templateMap.get(inspection.template_id)}
						<Table.Row class="cursor-pointer" onclick={() => onOpenInspection(inspection)}>
							<Table.Cell class="pl-4 text-xs">
								{formatIsoDateFr(inspection.inspection_date)}
							</Table.Cell>
							<Table.Cell>
								<Badge variant={inspection.type === 'MOVE_IN' ? 'secondary' : 'default'}>
									EDL {formatInspectionTypeLabel(inspection.type)}
								</Badge>
							</Table.Cell>
							<Table.Cell class="font-medium">
								{lease ? formatLeaseDisplayLabel(lease) : inspection.lease_id.slice(0, 8)}
							</Table.Cell>
							<Table.Cell class="text-xs text-muted-foreground">
								{template?.name ?? 'Grille standard'}
							</Table.Cell>
							<Table.Cell class="text-xs text-muted-foreground">
								{inspection.keys_handed_over_count} clé(s) · {inspection.items.length} élément(s)
							</Table.Cell>
							<Table.Cell class="pr-4 text-right">
								<div class="flex items-center justify-end gap-1.5">
									{#if inspection.report_download_url}
										<Button
											size="sm"
											variant="outline"
											href={inspection.report_download_url}
											target="_blank"
											rel="noreferrer"
											onclick={(event: MouseEvent) => {
												event.stopPropagation();
											}}
										>
											<IconDownload data-icon="inline-start" />
											PDF
										</Button>
									{/if}
									<Button
										size="sm"
										variant="secondary"
										onclick={(event: MouseEvent) => {
											event.stopPropagation();
											onOpenInspection(inspection);
										}}
									>
										Grille →
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
