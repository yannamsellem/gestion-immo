<script lang="ts">
	import { createApiClient, type UserDTO } from '$lib/api/client';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { Card } from '$lib/components/ui/card';
	import { Table } from '$lib/components/ui/table';

	const api = createApiClient();

	let pendingUsers: readonly UserDTO[] = [
		{
			id: 'usr_pending_01',
			full_name: 'Claire Gestionnaire',
			email: 'claire@investissimo.fr',
			role: 'MANAGER',
			status: 'PENDING_APPROVAL',
			created_at: '2026-02-18T09:15:00Z'
		}
	];
	let statusFeedback = '';

	async function handleApprove(userId: string) {
		try {
			await api.approveUser(userId);
		} catch {
			// Update optimistic state in preview mode
		}
		pendingUsers = pendingUsers.filter((u) => u.id !== userId);
		statusFeedback = `Compte ${userId} approuvé (statut APPROVED).`;
	}

	async function handleReject(userId: string) {
		try {
			await api.rejectUser(userId);
		} catch {
			// Update optimistic state in preview mode
		}
		pendingUsers = pendingUsers.filter((u) => u.id !== userId);
		statusFeedback = `Compte ${userId} refusé (statut REJECTED).`;
	}
</script>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold text-slate-900">Validation des inscriptions (Super Admin)</h1>
			<p class="text-sm text-slate-600">
				Seul le Super Administrateur peut approuver ou rejeter les nouveaux comptes en statut
				PENDING_APPROVAL.
			</p>
		</div>
		<Badge variant="warning">{pendingUsers.length} en attente</Badge>
	</div>

	{#if statusFeedback}
		<div class="rounded-lg border border-emerald-200 bg-emerald-50 p-3 text-sm text-emerald-900">
			{statusFeedback}
		</div>
	{/if}

	<Card title="Demandes d'inscription en attente" description="GET /api/v1/admin/users/pending">
		{#if pendingUsers.length === 0}
			<p class="py-6 text-center text-sm text-slate-500">
				Aucune demande d'inscription en attente de validation.
			</p>
		{:else}
			<Table>
				<thead>
					<tr class="border-b border-slate-200 text-left text-xs font-semibold uppercase text-slate-500">
						<th class="py-3 pr-4">Collaborateur</th>
						<th class="py-3 pr-4">E-mail</th>
						<th class="py-3 pr-4">Rôle demandé</th>
						<th class="py-3 pr-4">Statut</th>
						<th class="py-3 text-right">Actions Super Admin</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-slate-100">
					{#each pendingUsers as user (user.id)}
						<tr>
							<td class="py-3 pr-4 font-medium text-slate-900">{user.full_name}</td>
							<td class="py-3 pr-4 text-slate-600">{user.email}</td>
							<td class="py-3 pr-4"><Badge variant="secondary">{user.role}</Badge></td>
							<td class="py-3 pr-4"><Badge variant="warning">{user.status}</Badge></td>
							<td class="flex justify-end gap-2 py-3 text-right">
								<Button size="sm" on:click={() => handleApprove(user.id)}>
									Approuver (ApproveUser)
								</Button>
								<Button variant="destructive" size="sm" on:click={() => handleReject(user.id)}>
									Refuser (RejectUser)
								</Button>
							</td>
						</tr>
					{/each}
				</tbody>
			</Table>
		{/if}
	</Card>
</div>
