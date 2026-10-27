<script lang="ts">
	import { enhance } from '$app/forms';
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
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import * as Table from '#lib/components/ui/table/index.js';
	import { formatIsoDateFr } from '#lib/core/formatters/date';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	let mcpLabel = $state('Claude Desktop — Gestion Immo');
	let isCreatingMcp = $state(false);
	let processingUserId = $state<string | null>(null);

	$effect(() => {
		if (form?.successMessage) {
			toast.success(form.successMessage);
		}
		if (form?.actionError) {
			toast.error(form.actionError);
		}
	});

	async function copyTokenToClipboard(token: string) {
		if (typeof navigator !== 'undefined' && navigator.clipboard) {
			await navigator.clipboard.writeText(token);
			toast.success('Jeton MCP copié dans le presse-papiers.');
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
			<form
				method="POST"
				action="?/createMcpToken"
				use:enhance={() => {
					isCreatingMcp = true;
					return async ({ update }) => {
						await update({ reset: false });
						isCreatingMcp = false;
					};
				}}
				class="flex flex-col gap-3"
			>
				<Field.FieldGroup>
					<Field.Field>
						<Field.Label for="mcp-label">Libellé de l'intégration</Field.Label>
						<Input
							id="mcp-label"
							name="label"
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

			{#if form?.createdMcpToken}
				<Alert.Root>
					<IconKey class="size-4" />
					<Alert.Title>Jeton MCP généré ({form.createdMcpToken.label})</Alert.Title>
					<Alert.Description class="mt-2 flex flex-col gap-2">
						<p class="text-xs">
							Copiez ce jeton immédiatement. Par mesure de sécurité, il ne sera plus affiché en
							clair après fermeture de cette page :
						</p>
						<div class="flex items-center justify-between gap-2 rounded border bg-muted p-2">
							<code class="truncate font-mono text-[11px]">{form.createdMcpToken.token}</code>
							<Button
								size="sm"
								variant="secondary"
								onclick={() => copyTokenToClipboard(form.createdMcpToken!.token)}
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
			<Badge variant={data.isSuperAdmin ? 'default' : 'outline'}>
				{data.user?.role ?? 'MANAGER'}
			</Badge>
		</Card.Header>

		<Card.Content class="p-0">
			{#if !data.isSuperAdmin}
				<p class="px-4 py-6 text-xs text-muted-foreground">
					Votre compte dispose du rôle <strong>{data.user?.role ?? 'MANAGER'}</strong>. Seul un
					administrateur principal (<code>SUPER_ADMIN</code>) peut approuver ou rejeter les
					nouvelles demandes d'accès.
				</p>
			{:else if data.pendingError}
				<div class="p-4">
					<Alert.Root variant="destructive">
						<IconAlertTriangle class="size-4" />
						<Alert.Title>Erreur</Alert.Title>
						<Alert.Description>{data.pendingError}</Alert.Description>
					</Alert.Root>
				</div>
			{:else if data.pendingUsers.length === 0}
				<p class="px-4 py-6 text-xs text-muted-foreground">
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
						{#each data.pendingUsers as user (user.id)}
							<Table.Row>
								<Table.Cell class="pl-4 font-medium">{user.full_name}</Table.Cell>
								<Table.Cell class="text-xs">{user.email}</Table.Cell>
								<Table.Cell class="text-xs text-muted-foreground">
									{formatIsoDateFr(user.created_at)}
								</Table.Cell>
								<Table.Cell class="pr-4 text-right">
									<div class="flex items-center justify-end gap-1.5">
										<form
											method="POST"
											action="?/approveUser"
											use:enhance={() => {
												processingUserId = user.id;
												return async ({ update }) => {
													await update();
													processingUserId = null;
												};
											}}
										>
											<input type="hidden" name="user_id" value={user.id} />
											<input type="hidden" name="full_name" value={user.full_name} />
											<Button
												type="submit"
												size="sm"
												variant="default"
												disabled={processingUserId === user.id}
											>
												<IconCheck data-icon="inline-start" />
												Approuver
											</Button>
										</form>

										<form
											method="POST"
											action="?/rejectUser"
											use:enhance={() => {
												processingUserId = user.id;
												return async ({ update }) => {
													await update();
													processingUserId = null;
												};
											}}
										>
											<input type="hidden" name="user_id" value={user.id} />
											<input type="hidden" name="full_name" value={user.full_name} />
											<Button
												type="submit"
												size="sm"
												variant="destructive"
												disabled={processingUserId === user.id}
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
</div>
