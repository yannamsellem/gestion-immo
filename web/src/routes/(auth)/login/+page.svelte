<script lang="ts">
	import { goto } from '$app/navigation';
	import IconLock from '@tabler/icons-svelte/icons/lock';
	import { toast } from 'svelte-sonner';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import ThemeToggle from '#lib/components/layout/ThemeToggle.svelte';
	import { ApiError } from '#lib/core/api/api-error';
	import { authStore } from '#lib/modules/auth/auth-store.svelte';
	import { apiRegistry } from '#lib/modules/registry';

	let email = $state('');
	let password = $state('');
	let errorMessage = $state<string | null>(null);
	let isPendingApproval = $state(false);
	let isSubmitting = $state(false);

	async function handleSubmit(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		errorMessage = null;
		isPendingApproval = false;
		isSubmitting = true;

		try {
			const response = await apiRegistry.auth.login({
				email: email.trim(),
				password
			});
			authStore.setSession(response);
			toast.success(`Bienvenue, ${response.user.full_name}`);
			await goto('/');
		} catch (err) {
			if (err instanceof ApiError && err.status === 403) {
				isPendingApproval = true;
				errorMessage =
					"Votre compte est en attente de validation par un administrateur (SUPER_ADMIN).";
			} else if (err instanceof ApiError) {
				errorMessage = err.message;
			} else {
				errorMessage = 'Impossible de se connecter au serveur.';
			}
		} finally {
			isSubmitting = false;
		}
	}
</script>

<div class="bg-background flex min-h-screen flex-col justify-center p-4">
	<div class="fixed top-4 right-4">
		<ThemeToggle />
	</div>

	<Card.Root class="mx-auto w-full max-w-sm">
		<Card.Header>
			<Card.Title>Connexion à Gestion Immo</Card.Title>
			<Card.Description>
				Accédez à votre cockpit de gestion locative et comptable multi-entités.
			</Card.Description>
		</Card.Header>

		<Card.Content>
			{#if errorMessage}
				<Alert.Root variant={isPendingApproval ? 'default' : 'destructive'} class="mb-4">
					<Alert.Title>
						{isPendingApproval ? 'Approbation en attente' : 'Échec de connexion'}
					</Alert.Title>
					<Alert.Description>{errorMessage}</Alert.Description>
				</Alert.Root>
			{/if}

			<form onsubmit={handleSubmit}>
				<Field.FieldGroup>
					<Field.Field>
						<Field.FieldLabel for="login-email">Adresse e-mail</Field.FieldLabel>
						<Input
							id="login-email"
							type="email"
							required
							autocomplete="email"
							placeholder="gestionnaire@exemple.fr"
							bind:value={email}
						/>
					</Field.Field>

					<Field.Field>
						<Field.FieldLabel for="login-password">Mot de passe</Field.FieldLabel>
						<Input
							id="login-password"
							type="password"
							required
							autocomplete="current-password"
							placeholder="••••••••••••"
							bind:value={password}
						/>
					</Field.Field>

					<Button type="submit" class="w-full" disabled={isSubmitting}>
						{#if isSubmitting}
							<Spinner data-icon="inline-start" />
							Connexion en cours...
						{:else}
							<IconLock data-icon="inline-start" />
							Se connecter
						{/if}
					</Button>
				</Field.FieldGroup>
			</form>
		</Card.Content>

		<Card.Footer class="justify-center border-t pt-4">
			<p class="text-muted-foreground text-xs">
				Pas encore de compte ?
				<a href="/signup" class="text-primary font-medium underline-offset-4 hover:underline">
					Demander un accès
				</a>
			</p>
		</Card.Footer>
	</Card.Root>
</div>
