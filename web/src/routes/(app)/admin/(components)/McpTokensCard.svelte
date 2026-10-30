<script lang="ts">
	import { enhance } from '$app/forms';
	import IconAlertTriangle from '@tabler/icons-svelte/icons/alert-triangle';
	import IconCopy from '@tabler/icons-svelte/icons/copy';
	import IconKey from '@tabler/icons-svelte/icons/key';
	import { toast } from 'svelte-sonner';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import type { MCPTokenResponse } from '#lib/modules/auth/types';

	let mcpLabel = $state('Claude Desktop — Gestion Immo');
	let isCreating = $state(false);
	let createdToken = $state<MCPTokenResponse | null>(null);
	let errorMessage = $state<string | null>(null);

	async function copyTokenToClipboard(token: string): Promise<void> {
		if (typeof navigator !== 'undefined' && navigator.clipboard) {
			await navigator.clipboard.writeText(token);
			toast.success('Jeton MCP copié dans le presse-papiers.');
		}
	}
</script>

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
		{#if errorMessage}
			<Alert.Root variant="destructive">
				<IconAlertTriangle />
				<Alert.Title>Génération impossible</Alert.Title>
				<Alert.Description>{errorMessage}</Alert.Description>
			</Alert.Root>
		{/if}

		<form
			method="POST"
			action="?/createMcpToken"
			use:enhance={() => {
				isCreating = true;
				errorMessage = null;
				return async ({ result }) => {
					isCreating = false;
					if (result.type === 'success') {
						const payload = result.data as
							{ createdMcpToken?: MCPTokenResponse; successMessage?: string } | undefined;
						createdToken = payload?.createdMcpToken ?? null;
						toast.success(payload?.successMessage ?? 'Jeton personnel MCP généré.');
						return;
					}
					if (result.type === 'failure') {
						const payload = result.data as { actionError?: string } | undefined;
						const message = payload?.actionError ?? 'Échec de la création du jeton MCP.';
						errorMessage = message;
						toast.error(message);
						return;
					}
					if (result.type === 'error') {
						const message =
							result.error?.message ?? 'Erreur serveur lors de la création du jeton MCP.';
						errorMessage = message;
						toast.error(message);
					}
				};
			}}
			class="flex flex-col gap-3"
		>
			<Field.Group>
				<Field.Field>
					<Field.Label for="mcp-token-label">Libellé de l’intégration</Field.Label>
					<Input
						id="mcp-token-label"
						name="label"
						placeholder="Claude Desktop — MacBook Pro"
						bind:value={mcpLabel}
						required
					/>
				</Field.Field>
			</Field.Group>

			<Button type="submit" disabled={isCreating}>
				{#if isCreating}
					<Spinner data-icon="inline-start" />
				{/if}
				Générer un jeton MCP
			</Button>
		</form>

		{#if createdToken}
			<Alert.Root>
				<IconKey />
				<Alert.Title>Jeton MCP généré ({createdToken.label})</Alert.Title>
				<Alert.Description class="mt-2 flex flex-col gap-2">
					<p class="text-xs">
						Copiez ce jeton immédiatement. Par mesure de sécurité, il ne sera plus affiché en clair
						après fermeture de cette page :
					</p>
					<div class="flex items-center justify-between gap-2 rounded border bg-muted p-2">
						<code class="truncate font-mono text-[11px]">{createdToken.token}</code>
						<Button
							type="button"
							size="sm"
							variant="secondary"
							onclick={() => copyTokenToClipboard(createdToken!.token)}
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
