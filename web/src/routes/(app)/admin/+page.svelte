<script lang="ts">
	import IconAlertTriangle from '@tabler/icons-svelte/icons/alert-triangle';
	import IconCheck from '@tabler/icons-svelte/icons/check';
	import IconCopy from '@tabler/icons-svelte/icons/copy';
	import IconKey from '@tabler/icons-svelte/icons/key';
	import IconShieldCheck from '@tabler/icons-svelte/icons/shield-check';
	import IconX from '@tabler/icons-svelte/icons/x';
	import { toast } from 'svelte-sonner';
	import PageHeader from '#lib/components/layout/PageHeader.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import * as Table from '#lib/components/ui/table/index.js';
	import { ApiError } from '#lib/core/api/api-error';
	import { formatIsoDateFr } from '#lib/core/formatters/date';
	import { authStore } from '#lib/modules/auth/auth-store.svelte';
	import type { MCPTokenResponse, UserDTO } from '#lib/modules/auth/types';
	import { apiRegistry } from '#lib/modules/registry';

	let mcpLabel = $state('Claude Desktop — Gestion Immo');
	let isCreatingMcp = $state(false);
	let createdMcpToken = $state<MCPTokenResponse | null>(null);

	let pendingUsers = $state<UserDTO[]>([]);
	let isLoadingPending = $state(false);
	let pendingError = $state<string | null>(null);
	let processingUserId = $state<string | null>(null);

	$effect(() => {
		if (!authStore.isSuperAdmin) {
			pendingUsers = [];
			return;
		}

		let cancelled = false;
		isLoadingPending = true;
		pendingError = null;

		void apiRegistry.auth
			.listPendingUsers()
			.then((users) => {
				if (!cancelled) {
					pendingUsers = users;
				}
			})
			.catch((err) => {
				if (!cancelled) {
					pendingError =
						err instanceof ApiError
							? err.message
							: 'Impossible de charger les comptes en attente.';
				}
			})
			.finally(() => {
				if (!cancelled) {
					isLoadingPending = false;
				}
			});

		return () => {
			cancelled = true;
		};
	});

	async function handleCreateMcpToken(event: SubmitEvent) {
		event.preventDefault();
		if (!mcpLabel.trim()) return;

		isCreatingMcp = true;
		try {
			const response = await apiRegistry.auth.createMcpToken({ label: mcpLabel.trim() });
			createdMcpToken = response;
			toast.success('Jeton personnel MCP généré.');
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Échec de la création du jeton MCP.');
		} finally {
			isCreatingMcp = false;
		}
	}

	async function copyTokenToClipboard(token: string) {
		if (typeof navigator !== 'undefined' && navigator.clipboard) {
			await navigator.clipboard.writeText(token);
			toast.success('Jeton MCP copié dans le presse-papiers.');
		}
	}

	async function handleApproveUser(user: UserDTO) {
		processingUserId = user.id;
		try {
			await apiRegistry.auth.approveUser(user.id);
			pendingUsers = pendingUsers.filter((u) => u.id !== user.id);
			toast.success(`Compte de « ${user.full_name} » approuvé.`);
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Impossible d’approuver ce compte.');
		} finally {
			processingUserId = null;
		}
	}

	async function handleRejectUser(user: UserDTO) {
		processingUserId = user.id;
		try {
			await apiRegistry.auth.rejectUser(user.id);
			pendingUsers = pendingUsers.filter((u) => u.id !== user.id);
			toast.success(`Demande de « ${user.full_name} » rejetée.`);
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Impossible de rejeter ce compte.');
		} finally {
			processingUserId = null;
		}
	}
</script>

<PageHeader
	title="Administration, Sécurité & Accès MCP"
	subtitle="Génération de jetons d'accès MCP longue durée (Claude Desktop / Agents IA) et gouvernance des gestionnaires"
/>

<div class="grid gap-6 lg:grid-cols-12">
	<!-- Card 1: Personal MCP Tokens -->
	<Card.Root class="lg:col-span-5">
		<Card.Header class="border-b pb-3">
			<Card.Title class="flex items-center gap-1.5 text-sm font-semibold">
				<IconKey class="size-4" />
				Jetons MCP (Model Context Protocol)
			</Card.Title>
			<Card.Description>
				Générez un jeton personnel longue durée (ES256 / 1 an) pour connecter Claude Desktop ou un
				agent IA au serveur MCP <code>/mcp</code>.
			</Card.Description>
		</Card.Header>
		<Card.Content class="flex flex-col gap-4 pt-4">
			<form onsubmit={handleCreateMcpToken} class="flex flex-col gap-3">
				<Field.FieldGroup>
					<Field.Field>
						<Field.Label for="mcp-label">Libellé de l'intégration</Field.Label>
						<Input
							id="mcp-label"
							placeholder="Claude Desktop — MacBook Pro"
							bind:value={mcpLabel}
							required
						/>
					</Field.Field>
				</Field.FieldGroup>

				<Button type="submit" disabled={isCreatingMcp}>
					{#if isCreatingMcp}
						<Spinner data-icon="inline-start" />
					{/if}
					Générer un jeton MCP
				</Button>
			</form>

			{#if createdMcpToken}
				<Alert.Root>
					<IconKey class="size-4" />
					<Alert.Title>Jeton MCP généré ({createdMcpToken.label})</Alert.Title>
					<Alert.Description class="mt-2 flex flex-col gap-2">
						<p class="text-xs">
							Copiez ce jeton immédiatement. Par mesure de sécurité, il ne sera plus affiché en
							clair après fermeture de cette page :
						</p>
						<div class="bg-muted flex items-center justify-between gap-2 rounded border p-2">
							<code class="truncate font-mono text-[11px]">{createdMcpToken.token}</code>
							<Button
								size="sm"
								variant="secondary"
								onclick={() => copyTokenToClipboard(createdMcpToken!.token)}
							>
								<IconCopy data-icon="inline-start" />
								Copier
							</Button>
						</div>
					</Alert.Description>
				</Alert.Root>
			{/if}
		</Card.Content>
	</Card.Root>

	<!-- Card 2: SuperAdmin Pending Users Approval -->
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
			<Badge variant={authStore.isSuperAdmin ? 'default' : 'outline'}>
				{authStore.user?.role ?? 'MANAGER'}
			</Badge>
		</Card.Header>

		<Card.Content class="p-0">
			{#if !authStore.isSuperAdmin}
				<p class="text-muted-foreground px-4 py-6 text-xs">
					Votre compte dispose du rôle <strong>{authStore.user?.role ?? 'MANAGER'}</strong>. Seul un
					administrateur principal (<code>SUPER_ADMIN</code>) peut approuver ou rejeter les
					nouvelles demandes d'accès.
				</p>
			{:else if pendingError}
				<div class="p-4">
					<Alert.Root variant="destructive">
						<IconAlertTriangle class="size-4" />
						<Alert.Title>Erreur</Alert.Title>
						<Alert.Description>{pendingError}</Alert.Description>
					</Alert.Root>
				</div>
			{:else if isLoadingPending}
				<div class="p-4">
					<Skeleton class="h-24 w-full" />
				</div>
			{:else if pendingUsers.length === 0}
				<p class="text-muted-foreground px-4 py-6 text-xs">
					Aucune demande d'inscription en attente d'approbation.
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
						{#each pendingUsers as user (user.id)}
							<Table.Row>
								<Table.Cell class="pl-4 font-medium">{user.full_name}</Table.Cell>
								<Table.Cell class="text-xs">{user.email}</Table.Cell>
								<Table.Cell class="text-muted-foreground text-xs">
									{formatIsoDateFr(user.created_at)}
								</Table.Cell>
								<Table.Cell class="pr-4 text-right">
									<div class="flex items-center justify-end gap-1.5">
										<Button
											size="sm"
											variant="default"
											disabled={processingUserId === user.id}
											onclick={() => handleApproveUser(user)}
										>
											<IconCheck data-icon="inline-start" />
											Approuver
										</Button>
										<Button
											size="sm"
											variant="destructive"
											disabled={processingUserId === user.id}
											onclick={() => handleRejectUser(user)}
										>
											<IconX data-icon="inline-start" />
											Rejeter
										</Button>
									</div>
								</Table.Cell>
							</Table.Row>
						{/each}
					</Table.Body>
				</Table.Root>
			{/if}
		</Card.Content>
	</Card.Root>
</div>
