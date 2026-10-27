<script lang="ts">
	import IconUserPlus from '@tabler/icons-svelte/icons/user-plus';
	import { toast } from 'svelte-sonner';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import ThemeToggle from '#lib/components/layout/ThemeToggle.svelte';
	import { ApiError } from '#lib/core/api/api-error';
	import { apiRegistry } from '#lib/modules/registry';

	let fullName = $state('');
	let email = $state('');
	let password = $state('');
	let errorMessage = $state<string | null>(null);
	let submittedEmail = $state<string | null>(null);
	let isSubmitting = $state(false);

	async function handleSubmit(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		errorMessage = null;
		isSubmitting = true;

		try {
			const created = await apiRegistry.auth.signup({
				full_name: fullName.trim(),
				email: email.trim(),
				password
			});
			submittedEmail = created.email;
			toast.success('Demande de compte enregistrée');
		} catch (err) {
			if (err instanceof ApiError) {
				errorMessage = err.message;
			} else {
				errorMessage = "Impossible d'enregistrer votre demande.";
			}
		} finally {
			isSubmitting = false;
		}
	}
</script>

<div class="flex min-h-screen flex-col justify-center bg-background p-4">
	<div class="fixed top-4 right-4">
		<ThemeToggle />
	</div>

	<Card.Root class="mx-auto w-full max-w-sm">
		<Card.Header>
			<Card.Title>Créer un compte gestionnaire</Card.Title>
			<Card.Description>
				Les nouveaux comptes sont soumis à validation par un administrateur principal.
			</Card.Description>
		</Card.Header>

		<Card.Content>
			{#if submittedEmail}
				<Alert.Root>
					<Alert.Title>Compte en attente d'approbation (202 Accepted)</Alert.Title>
					<Alert.Description>
						Votre demande pour <strong>{submittedEmail}</strong> a bien été enregistrée avec le
						statut
						<code>PENDING_APPROVAL</code>. Un administrateur doit l'approuver avant votre première
						connexion.
					</Alert.Description>
				</Alert.Root>
			{:else}
				{#if errorMessage}
					<Alert.Root variant="destructive" class="mb-4">
						<Alert.Title>Erreur d'inscription</Alert.Title>
						<Alert.Description>{errorMessage}</Alert.Description>
					</Alert.Root>
				{/if}

				<form onsubmit={handleSubmit}>
					<Field.FieldGroup>
						<Field.Field>
							<Field.FieldLabel for="signup-name">Nom complet</Field.FieldLabel>
							<Input
								id="signup-name"
								type="text"
								required
								placeholder="Jean Dupont"
								bind:value={fullName}
							/>
						</Field.Field>

						<Field.Field>
							<Field.FieldLabel for="signup-email">Adresse e-mail professionnelle</Field.FieldLabel>
							<Input
								id="signup-email"
								type="email"
								required
								autocomplete="email"
								placeholder="jean.dupont@exemple.fr"
								bind:value={email}
							/>
						</Field.Field>

						<Field.Field>
							<Field.FieldLabel for="signup-password">
								Mot de passe (8 caractères minimum)
							</Field.FieldLabel>
							<Input
								id="signup-password"
								type="password"
								required
								minlength={8}
								autocomplete="new-password"
								placeholder="••••••••••••"
								bind:value={password}
							/>
						</Field.Field>

						<Button type="submit" class="w-full" disabled={isSubmitting}>
							{#if isSubmitting}
								<Spinner data-icon="inline-start" />
								Envoi en cours...
							{:else}
								<IconUserPlus data-icon="inline-start" />
								Envoyer la demande
							{/if}
						</Button>
					</Field.FieldGroup>
				</form>
			{/if}
		</Card.Content>

		<Card.Footer class="justify-center border-t pt-4">
			<p class="text-xs text-muted-foreground">
				Déjà approuvé ?
				<a href="/login" class="font-medium text-primary underline-offset-4 hover:underline">
					Retour à la connexion
				</a>
			</p>
		</Card.Footer>
	</Card.Root>
</div>
