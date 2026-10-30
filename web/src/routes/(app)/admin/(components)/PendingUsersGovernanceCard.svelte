<script lang="ts">
	import { enhance } from '$app/forms';
	import { invalidateAll } from '$app/navigation';
	import IconAlertTriangle from '@tabler/icons-svelte/icons/alert-triangle';
	import IconCheck from '@tabler/icons-svelte/icons/check';
	import IconShieldCheck from '@tabler/icons-svelte/icons/shield-check';
	import IconX from '@tabler/icons-svelte/icons/x';
	import { toast } from 'svelte-sonner';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Table from '#lib/components/ui/table/index.js';
	import { formatIsoDateFr } from '#lib/core/formatters/date';
	import type { UserDTO } from '#lib/modules/auth/types';

	let {
		isSuperAdmin,
		userRole,
		pendingUsers,
		pendingError
	}: {
		isSuperAdmin: boolean;
		userRole: string;
		pendingUsers: UserDTO[];
		pendingError: string | null;
	} = $props();

	let processingUserId = $state<string | null>(null);
</script>

<Card.Root class="lg:col-span-7">
	<Card.Header class="flex flex-row items-center justify-between border-b pb-3">
		<div>
			<Card.Title class="flex items-center gap-1.5 text-sm font-semibold">
				<IconShieldCheck class="size-4" />
				Gouvernance des accès & Approbations
			</Card.Title>
			<Card.Description>
				Validation manuelle des inscriptions en attente (réservé au rôle SUPER_ADMIN)
			</Card.Description>
		</div>
		<Badge variant={isSuperAdmin ? 'default' : 'outline'}>
			{userRole}
		</Badge>
	</Card.Header>

	<Card.Content class="p-0">
		{#if !isSuperAdmin}
			<p class="px-4 py-6 text-xs text-muted-foreground">
				Votre compte dispose du rôle <strong>{userRole}</strong>. Seul un administrateur principal (<code
					>SUPER_ADMIN</code
				>) peut approuver ou rejeter les nouvelles demandes d’accès.
			</p>
		{:else if pendingError}
			<div class="p-4">
				<Alert.Root variant="destructive">
					<IconAlertTriangle />
					<Alert.Title>Erreur</Alert.Title>
					<Alert.Description>{pendingError}</Alert.Description>
				</Alert.Root>
			</div>
		{:else if pendingUsers.length === 0}
			<p class="px-4 py-6 text-xs text-muted-foreground">
				Aucune demande d’inscription en attente d’approbation.
			</p>
		{:else}
			<Table.Root>
				<Table.Header>
					<Table.Row>
						<Table.Head class="pl-4">Collaborateur</Table.Head>
						<Table.Head>Email</Table.Head>
						<Table.Head>Demandé le</Table.Head>
						<Table.Head class="pr-4 text-right">Décision</Table.Head>
					</Table.Row>
				</Table.Header>
				<Table.Body>
					{#each pendingUsers as pendingUser (pendingUser.id)}
						<Table.Row>
							<Table.Cell class="pl-4 font-medium">{pendingUser.full_name}</Table.Cell>
							<Table.Cell class="text-xs">{pendingUser.email}</Table.Cell>
							<Table.Cell class="text-xs text-muted-foreground">
								{formatIsoDateFr(pendingUser.created_at)}
							</Table.Cell>
							<Table.Cell class="pr-4 text-right">
								<div class="flex items-center justify-end gap-1.5">
									<form
										method="POST"
										action="?/approveUser"
										use:enhance={() => {
											processingUserId = pendingUser.id;
											return async ({ result }) => {
												processingUserId = null;
												if (result.type === 'success') {
													const payload = result.data as { successMessage?: string } | undefined;
													toast.success(payload?.successMessage ?? 'Compte approuvé.');
													await invalidateAll();
													return;
												}
												if (result.type === 'failure') {
													const payload = result.data as { actionError?: string } | undefined;
													toast.error(payload?.actionError ?? 'Impossible d’approuver ce compte.');
													return;
												}
												if (result.type === 'error') {
													toast.error(
														result.error?.message ?? 'Erreur serveur lors de l’approbation.'
													);
												}
											};
										}}
									>
										<input type="hidden" name="user_id" value={pendingUser.id} />
										<input type="hidden" name="full_name" value={pendingUser.full_name} />
										<Button
											type="submit"
											size="sm"
											variant="default"
											disabled={processingUserId === pendingUser.id}
										>
											<IconCheck data-icon="inline-start" />
											Approuver
										</Button>
									</form>

									<form
										method="POST"
										action="?/rejectUser"
										use:enhance={() => {
											processingUserId = pendingUser.id;
											return async ({ result }) => {
												processingUserId = null;
												if (result.type === 'success') {
													const payload = result.data as { successMessage?: string } | undefined;
													toast.success(payload?.successMessage ?? 'Demande rejetée.');
													await invalidateAll();
													return;
												}
												if (result.type === 'failure') {
													const payload = result.data as { actionError?: string } | undefined;
													toast.error(payload?.actionError ?? 'Impossible de rejeter ce compte.');
													return;
												}
												if (result.type === 'error') {
													toast.error(result.error?.message ?? 'Erreur serveur lors du rejet.');
												}
											};
										}}
									>
										<input type="hidden" name="user_id" value={pendingUser.id} />
										<input type="hidden" name="full_name" value={pendingUser.full_name} />
										<Button
											type="submit"
											size="sm"
											variant="destructive"
											disabled={processingUserId === pendingUser.id}
										>
											<IconX data-icon="inline-start" />
											Rejeter
										</Button>
									</form>
								</div>
							</Table.Cell>
						</Table.Row>
					{/each}
				</Table.Body>
			</Table.Root>
		{/if}
	</Card.Content>
</Card.Root>
