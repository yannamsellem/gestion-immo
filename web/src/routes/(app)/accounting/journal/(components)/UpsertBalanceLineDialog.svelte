<script lang="ts">
	import { enhance } from '$app/forms';
	import { invalidateAll } from '$app/navigation';
	import IconAlertTriangle from '@tabler/icons-svelte/icons/alert-triangle';
	import { toast } from 'svelte-sonner';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Dialog from '#lib/components/ui/dialog/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import * as Select from '#lib/components/ui/select/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import {
		STATEMENT_SECTION_OPTIONS,
		formatStatementSectionLabel
	} from '#lib/modules/accounting/statement-state';
	import type { StatementSection } from '#lib/modules/accounting/types';

	let {
		open = $bindable(false),
		activeEntityId,
		fiscalYear,
		existingLinesJson
	}: {
		open?: boolean;
		activeEntityId: string;
		fiscalYear: number;
		existingLinesJson: string;
	} = $props();

	let accountCode = $state('706000');
	let label = $state('Loyers et revenus locatifs');
	let section = $state<StatementSection>('OPERATING_REVENUE');
	let cerfaCode = $state('2072-R1');
	let netNEuro = $state('18000,00');
	let netNMinus1Euro = $state('16500,00');
	let isSubmitting = $state(false);
	let errorMessage = $state<string | null>(null);

	function resetForm(): void {
		accountCode = '706000';
		label = 'Loyers et revenus locatifs';
		section = 'OPERATING_REVENUE';
		cerfaCode = '2072-R1';
		netNEuro = '18000,00';
		netNMinus1Euro = '16500,00';
		errorMessage = null;
	}
</script>

<Dialog.Root
	bind:open
	onOpenChange={(next) => {
		if (!next) {
			errorMessage = null;
		}
	}}
>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>Ajouter / modifier un poste de balance</Dialog.Title>
			<Dialog.Description>
				Alimentez un compte PCG (ex. 706000 Loyers, 681100 Amortissements, 213000 Immeubles) pour
				l’exercice {fiscalYear}.
			</Dialog.Description>
		</Dialog.Header>

		{#if errorMessage}
			<Alert.Root variant="destructive">
				<IconAlertTriangle />
				<Alert.Title>Enregistrement impossible</Alert.Title>
				<Alert.Description>{errorMessage}</Alert.Description>
			</Alert.Root>
		{/if}

		<form
			method="POST"
			action="?/upsertBalanceLine"
			use:enhance={() => {
				isSubmitting = true;
				errorMessage = null;
				return async ({ result }) => {
					isSubmitting = false;
					if (result.type === 'success') {
						const payload = result.data as { successMessage?: string } | undefined;
						toast.success(payload?.successMessage ?? 'Ligne de balance enregistrée.');
						resetForm();
						open = false;
						await invalidateAll();
						return;
					}
					if (result.type === 'failure') {
						const payload = result.data as { actionError?: string } | undefined;
						const message = payload?.actionError ?? 'Impossible d’enregistrer la ligne de balance.';
						errorMessage = message;
						toast.error(message);
						return;
					}
					if (result.type === 'error') {
						const message =
							result.error?.message ?? 'Erreur serveur lors de la mise à jour de la balance.';
						errorMessage = message;
						toast.error(message);
					}
				};
			}}
			class="flex flex-col gap-4"
		>
			<input type="hidden" name="entity_id" value={activeEntityId} />
			<input type="hidden" name="fiscal_year" value={fiscalYear} />
			<input type="hidden" name="existing_lines_json" value={existingLinesJson} />
			<input type="hidden" name="section" value={section} />

			<Field.Group>
				<div class="grid gap-3 sm:grid-cols-2">
					<Field.Field>
						<Field.Label for="bal-code">N° de compte PCG</Field.Label>
						<Input
							id="bal-code"
							name="account_code"
							class="font-mono"
							bind:value={accountCode}
							required
						/>
					</Field.Field>

					<Field.Field>
						<Field.Label for="bal-cerfa">Case CERFA (2072 / 2033)</Field.Label>
						<Input
							id="bal-cerfa"
							name="cerfa_code"
							placeholder="2072-R1 / 2033-A"
							bind:value={cerfaCode}
						/>
					</Field.Field>
				</div>

				<Field.Field>
					<Field.Label for="bal-label">Intitulé du compte</Field.Label>
					<Input id="bal-label" name="label" bind:value={label} required />
				</Field.Field>

				<Field.Field>
					<Field.Label>Rubrique des états financiers</Field.Label>
					<Select.Root
						type="single"
						value={section}
						onValueChange={(next) => {
							if (next) section = next;
						}}
					>
						<Select.Trigger class="w-full">
							{formatStatementSectionLabel(section)}
						</Select.Trigger>
						<Select.Content>
							<Select.Group>
								{#each STATEMENT_SECTION_OPTIONS as option (option.value)}
									<Select.Item value={option.value} label={option.label}>
										{option.label}
									</Select.Item>
								{/each}
							</Select.Group>
						</Select.Content>
					</Select.Root>
				</Field.Field>

				<div class="grid gap-3 sm:grid-cols-2">
					<Field.Field>
						<Field.Label for="bal-n">Montant Net {fiscalYear} (€)</Field.Label>
						<Input
							id="bal-n"
							name="net_n_euro"
							type="text"
							inputmode="decimal"
							bind:value={netNEuro}
							required
						/>
					</Field.Field>

					<Field.Field>
						<Field.Label for="bal-n1">Montant Net {fiscalYear - 1} (€)</Field.Label>
						<Input
							id="bal-n1"
							name="net_n_minus_1_euro"
							type="text"
							inputmode="decimal"
							bind:value={netNMinus1Euro}
							required
						/>
					</Field.Field>
				</div>
			</Field.Group>

			<Dialog.Footer>
				<Button
					type="button"
					variant="outline"
					disabled={isSubmitting}
					onclick={() => {
						open = false;
					}}
				>
					Annuler
				</Button>
				<Button type="submit" disabled={isSubmitting || !activeEntityId}>
					{#if isSubmitting}
						<Spinner data-icon="inline-start" />
					{/if}
					Enregistrer le poste
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
