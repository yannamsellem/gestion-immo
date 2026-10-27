<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import IconLock from '@tabler/icons-svelte/icons/lock';
	import ThemeToggle from '#lib/components/layout/ThemeToggle.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import type { PageProps } from './$types';

	let { form }: PageProps = $props();
	let isSubmitting = $state(false);
</script>

<div class="flex min-h-screen flex-col justify-center bg-background p-4">
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
			{#if form?.errorMessage}
				<Alert.Root variant={form.isPendingApproval ? 'default' : 'destructive'} class="mb-4">
					<Alert.Title>
						{form.isPendingApproval ? 'Approbation en attente' : 'Échec de connexion'}
					</Alert.Title>
					<Alert.Description>{form.errorMessage}</Alert.Description>
				</Alert.Root>
			{/if}

			<form
				method="POST"
				use:enhance={() => {
					isSubmitting = true;
					return async ({ update }) => {
						await update();
						isSubmitting = false;
					};
				}}
			>
				<Field.FieldGroup>
					<Field.Field>
						<Field.FieldLabel for="login-email">Adresse e-mail</Field.FieldLabel>
						<Input
							id="login-email"
							name="email"
							type="email"
							required
							autocomplete="email"
							placeholder="gestionnaire@exemple.fr"
							value={form?.email ?? ''}
						/>
					</Field.Field>

					<Field.Field>
						<Field.FieldLabel for="login-password">Mot de passe</Field.FieldLabel>
						<Input
							id="login-password"
							name="password"
							type="password"
							required
							autocomplete="current-password"
							placeholder="••••••••••••"
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
			<p class="text-xs text-muted-foreground">
				Pas encore de compte ?
				<a
					href={resolve('/(auth)/signup')}
					class="font-medium text-primary underline-offset-4 hover:underline"
				>
					Demander un accès
				</a>
			</p>
		</Card.Footer>
	</Card.Root>
</div>
