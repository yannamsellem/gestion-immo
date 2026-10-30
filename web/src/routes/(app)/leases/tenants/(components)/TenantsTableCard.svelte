<script lang="ts">
	import { enhance } from '$app/forms';
	import { invalidateAll } from '$app/navigation';
	import IconArchive from '@tabler/icons-svelte/icons/archive';
	import IconUserPlus from '@tabler/icons-svelte/icons/user-plus';
	import IconUsers from '@tabler/icons-svelte/icons/users';
	import { toast } from 'svelte-sonner';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Empty from '#lib/components/ui/empty/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import * as Table from '#lib/components/ui/table/index.js';
	import type { TenantDTO } from '#lib/modules/lease/types';

	let {
		tenants,
		activeLeaseCountByTenantId,
		onCreateTenant
	}: {
		tenants: TenantDTO[];
		activeLeaseCountByTenantId: Record<string, number>;
		onCreateTenant: () => void;
	} = $props();

	let archivingTenantId = $state<string | null>(null);
</script>

{#if tenants.length === 0}
	<Card.Root>
		<Card.Content class="py-10">
			<Empty.Root>
				<Empty.Header>
					<Empty.Media variant="icon">
						<IconUsers class="size-5" />
					</Empty.Media>
					<Empty.Title>Aucun locataire enregistré</Empty.Title>
					<Empty.Description>
						Créez une fiche locataire pour l’associer ensuite à un contrat de bail locatif.
					</Empty.Description>
				</Empty.Header>
				<Empty.Content>
					<Button onclick={onCreateTenant}>
						<IconUserPlus data-icon="inline-start" />
						Nouveau locataire
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
						<Table.Head class="pl-4">Nom complet</Table.Head>
						<Table.Head>Email & Téléphone</Table.Head>
						<Table.Head>Adresse</Table.Head>
						<Table.Head>Baux actifs</Table.Head>
						<Table.Head>Statut RGPD</Table.Head>
						<Table.Head class="pr-4 text-right">Action</Table.Head>
					</Table.Row>
				</Table.Header>
				<Table.Body>
					{#each tenants as tenant (tenant.id)}
						{@const activeLeaseCount = activeLeaseCountByTenantId[tenant.id] ?? 0}
						{@const isArchiving = archivingTenantId === tenant.id}
						<Table.Row>
							<Table.Cell class="pl-4 font-medium">{tenant.full_name}</Table.Cell>
							<Table.Cell class="text-xs">
								<span>{tenant.email}</span>
								{#if tenant.phone}
									<span class="block text-muted-foreground">{tenant.phone}</span>
								{/if}
							</Table.Cell>
							<Table.Cell class="text-xs text-muted-foreground">
								{tenant.address || '—'}
							</Table.Cell>
							<Table.Cell>
								{#if activeLeaseCount > 0}
									<Badge variant="secondary">
										{activeLeaseCount} bail{activeLeaseCount > 1 ? 'x' : ''} actif{activeLeaseCount >
										1
											? 's'
											: ''}
									</Badge>
								{:else}
									<span class="text-xs text-muted-foreground">Aucun</span>
								{/if}
							</Table.Cell>
							<Table.Cell>
								{#if tenant.archived}
									<Badge variant="outline">Archivé RGPD</Badge>
								{:else}
									<Badge variant="secondary">Actif</Badge>
								{/if}
							</Table.Cell>
							<Table.Cell class="pr-4 text-right">
								{#if !tenant.archived}
									<form
										method="POST"
										action="?/archiveTenant"
										use:enhance={() => {
											archivingTenantId = tenant.id;
											return async ({ result }) => {
												archivingTenantId = null;
												if (result.type === 'success') {
													const payload = result.data as { successMessage?: string } | undefined;
													toast.success(
														payload?.successMessage ?? 'Locataire archivé (conformité RGPD).'
													);
													await invalidateAll();
													return;
												}
												if (result.type === 'failure') {
													const payload = result.data as { actionError?: string } | undefined;
													toast.error(
														payload?.actionError ?? 'Impossible d’archiver ce locataire.'
													);
													return;
												}
												if (result.type === 'error') {
													toast.error(
														result.error?.message ?? 'Erreur serveur lors de l’archivage RGPD.'
													);
												}
											};
										}}
										class="inline"
									>
										<input type="hidden" name="tenant_id" value={tenant.id} />
										<input type="hidden" name="tenant_name" value={tenant.full_name} />
										<Button type="submit" size="sm" variant="outline" disabled={isArchiving}>
											{#if isArchiving}
												<Spinner data-icon="inline-start" />
											{:else}
												<IconArchive data-icon="inline-start" />
											{/if}
											Archiver RGPD
										</Button>
									</form>
								{/if}
							</Table.Cell>
						</Table.Row>
					{/each}
				</Table.Body>
			</Table.Root>
		</Card.Content>
	</Card.Root>
{/if}
