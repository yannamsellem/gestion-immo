<script lang="ts">
	import { createApiClient, type UserDTO } from '$lib/api/client';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { Card } from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';

	let fullName = '';
	let email = '';
	let password = '';
	let loading = false;
	let createdUser: UserDTO | null = null;
	let errorMessage = '';

	const api = createApiClient();

	async function handleSignupSubmit(event: SubmitEvent) {
		event.preventDefault();
		loading = true;
		errorMessage = '';

		try {
			createdUser = await api.signup({
				full_name: fullName,
				email,
				password
			});
		} catch (err) {
			errorMessage = err instanceof Error ? err.message : "Erreur lors de l'inscription.";
		} finally {
			loading = false;
		}
	}
</script>

<div class="mx-auto max-w-md py-12">
	<Card
		title="Demande d'ouverture de compte"
		description="Toute nouvelle inscription est placée en statut PENDING_APPROVAL jusqu'à validation par le Super Administrateur."
	>
		{#if createdUser}
			<div class="space-y-3 rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
				<div class="flex items-center justify-between">
					<span class="font-semibold">{createdUser.full_name} ({createdUser.email})</span>
					<Badge variant="warning">{createdUser.status}</Badge>
				</div>
				<p>
					Votre inscription est en attente de validation par le Super Administrateur. Aucun jeton
					d'accès n'est émis tant que le compte n'a pas été approuvé.
				</p>
				<div class="pt-2">
					<a href="/login" class="font-medium text-blue-700 hover:underline">
						Retour à la page de connexion
					</a>
				</div>
			</div>
		{:else}
			{#if errorMessage}
				<div class="mb-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-800">
					{errorMessage}
				</div>
			{/if}

			<form on:submit={handleSignupSubmit} class="space-y-4">
				<div>
					<label for="signup-name" class="mb-1 block text-sm font-medium text-slate-700">
						Nom complet
					</label>
					<Input
						id="signup-name"
						placeholder="Collaborateur Gestion"
						bind:value={fullName}
						required
					/>
				</div>

				<div>
					<label for="signup-email" class="mb-1 block text-sm font-medium text-slate-700">
						Adresse e-mail
					</label>
					<Input
						id="signup-email"
						type="email"
						placeholder="gestionnaire@investissimo.fr"
						bind:value={email}
						required
					/>
				</div>

				<div>
					<label for="signup-password" class="mb-1 block text-sm font-medium text-slate-700">
						Mot de passe (12 caractères min.)
					</label>
					<Input
						id="signup-password"
						type="password"
						placeholder="••••••••••••"
						bind:value={password}
						required
					/>
				</div>

				<div class="flex items-center justify-between pt-2">
					<a href="/login" class="text-sm text-slate-600 hover:underline">Déjà inscrit ?</a>
					<Button type="submit" disabled={loading}>
						{loading ? 'Envoi...' : "Soumettre l'inscription"}
					</Button>
				</div>
			</form>
		{/if}
	</Card>
</div>
