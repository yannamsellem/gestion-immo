<script lang="ts">
	import { ApiError, createApiClient } from '$lib/api/client';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { Card } from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';

	let email = '';
	let password = '';
	let loading = false;
	let pendingApprovalNotice = false;
	let errorMessage = '';
	let authenticatedUserEmail = '';

	const api = createApiClient();

	async function handleLoginSubmit(event: SubmitEvent) {
		event.preventDefault();
		loading = true;
		pendingApprovalNotice = false;
		errorMessage = '';

		try {
			const result = await api.login({ email, password });
			authenticatedUserEmail = result.user.email;
		} catch (err) {
			if (
				err instanceof ApiError &&
				(err.status === 403 || err.code === 'ACCOUNT_NOT_APPROVED')
			) {
				pendingApprovalNotice = true;
			} else if (err instanceof Error) {
				errorMessage = err.message;
			} else {
				errorMessage = 'Identifiants invalides ou serveur indisponible.';
			}
		} finally {
			loading = false;
		}
	}
</script>

<div class="mx-auto max-w-md py-12">
	<Card
		title="Connexion au Back-Office Gestion Immo"
		description="Accès sécurisé réservé aux comptes validés par le Super Administrateur."
	>
		{#if pendingApprovalNotice}
			<div
				class="mb-4 rounded-lg border border-amber-300 bg-amber-50 p-4 text-sm text-amber-900"
				role="alert"
			>
				<div class="flex items-center gap-2 font-semibold">
					<Badge variant="warning">403 ACCOUNT_NOT_APPROVED</Badge>
					<span>Compte en attente</span>
				</div>
				<p class="mt-2">
					Votre inscription est en attente de validation par le Super Administrateur
				</p>
			</div>
		{/if}

		{#if errorMessage}
			<div
				class="mb-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-800"
				role="alert"
			>
				{errorMessage}
			</div>
		{/if}

		{#if authenticatedUserEmail}
			<div class="mb-4 rounded-lg border border-emerald-200 bg-emerald-50 p-3 text-sm text-emerald-800">
				Connecté en tant que <strong>{authenticatedUserEmail}</strong>.
			</div>
		{/if}

		<form on:submit={handleLoginSubmit} class="space-y-4">
			<div>
				<label for="login-email" class="mb-1 block text-sm font-medium text-slate-700">
					Adresse e-mail professionnelle
				</label>
				<Input
					id="login-email"
					type="email"
					placeholder="yann@investissimo.fr"
					bind:value={email}
					required
				/>
			</div>

			<div>
				<label for="login-password" class="mb-1 block text-sm font-medium text-slate-700">
					Mot de passe
				</label>
				<Input
					id="login-password"
					type="password"
					placeholder="••••••••••••"
					bind:value={password}
					required
				/>
			</div>

			<div class="flex items-center justify-between pt-2">
				<a href="/signup" class="text-sm text-blue-600 hover:underline">
					Demander une ouverture de compte
				</a>
				<Button type="submit" disabled={loading}>
					{loading ? 'Connexion...' : 'Se connecter'}
				</Button>
			</div>
		</form>
	</Card>
</div>
