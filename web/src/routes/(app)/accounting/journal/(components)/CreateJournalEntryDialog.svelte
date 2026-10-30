<script lang="ts">
	import { enhance } from '$app/forms';
	import { invalidateAll } from '$app/navigation';
	import IconAlertTriangle from '@tabler/icons-svelte/icons/alert-triangle';
	import { toast } from 'svelte-sonner';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Dialog from '#lib/components/ui/dialog/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import * as Select from '#lib/components/ui/select/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import { formatCentsToEuro } from '#lib/core/formatters/money';
	import {
		STATEMENT_SECTION_OPTIONS,
		evaluateJournalBalance,
		formatStatementSectionLabel
	} from '#lib/modules/accounting/statement-state';
	import type { StatementSection } from '#lib/modules/accounting/types';

	let {
		open = $bindable(false),
		activeEntityId,
		fiscalYear
	}: {
		open?: boolean;
		activeEntityId: string;
		fiscalYear: number;
	} = $props();

	let journalCode = $state('OD');
	let journalLabel = $state('Opérations diverses');
	let documentRef = $state('FACT-2026-01');
	let entryLabel = $state('Dotation aux amortissements / Charge locative');
	let debitAccount = $state('681100');
	let debitAccountLabel = $state('Dotations aux amortissements sur immobilisations');
	let debitSection = $state<StatementSection>('DEPRECIATION');
	let debitEuro = $state('1200,00');
	let creditAccount = $state('281300');
	let creditAccountLabel = $state('Amortissements des constructions');
	let creditSection = $state<StatementSection>('FIXED_ASSETS');
	let creditEuro = $state('1200,00');
	let isSubmitting = $state(false);
	let errorMessage = $state<string | null>(null);

	const journalBalanceCheck = $derived(evaluateJournalBalance(debitEuro, creditEuro));

	function resetForm(): void {
		journalCode = 'OD';
		journalLabel = 'Opérations diverses';
		documentRef = 'FACT-2026-01';
		entryLabel = 'Dotation aux amortissements / Charge locative';
		debitAccount = '681100';
		debitAccountLabel = 'Dotations aux amortissements sur immobilisations';
		debitSection = 'DEPRECIATION';
		debitEuro = '1200,00';
		creditAccount = '281300';
		creditAccountLabel = 'Amortissements des constructions';
		creditSection = 'FIXED_ASSETS';
		creditEuro = '1200,00';
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
	<Dialog.Content class="max-h-[90vh] overflow-y-auto sm:max-w-lg">
		<Dialog.Header>
			<Dialog.Title>Saisir une écriture comptable en partie double</Dialog.Title>
			<Dialog.Description>
				L’équilibre strict Débit = Crédit est vérifié avant l’inscription au livre-journal FEC de
				l’exercice {fiscalYear}.
			</Dialog.Description>
		</Dialog.Header>

		{#if errorMessage}
			<Alert.Root variant="destructive">
				<IconAlertTriangle />
				<Alert.Title>Validation impossible</Alert.Title>
				<Alert.Description>{errorMessage}</Alert.Description>
			</Alert.Root>
		{/if}

		<form
			method="POST"
			action="?/recordJournalEntry"
			use:enhance={() => {
				isSubmitting = true;
				errorMessage = null;
				return async ({ result }) => {
					isSubmitting = false;
					if (result.type === 'success') {
						const payload = result.data as { successMessage?: string } | undefined;
						toast.success(payload?.successMessage ?? 'Écriture comptable validée.');
						resetForm();
						open = false;
						await invalidateAll();
						return;
					}
					if (result.type === 'failure') {
						const payload = result.data as { actionError?: string } | undefined;
						const message = payload?.actionError ?? 'Impossible d’enregistrer l’écriture.';
						errorMessage = message;
						toast.error(message);
						return;
					}
					if (result.type === 'error') {
						const message =
							result.error?.message ?? 'Erreur serveur lors de l’enregistrement au livre-journal.';
						errorMessage = message;
						toast.error(message);
					}
				};
			}}
			class="flex flex-col gap-4"
		>
			<input type="hidden" name="entity_id" value={activeEntityId} />
			<input type="hidden" name="fiscal_year" value={fiscalYear} />
			<input type="hidden" name="debit_section" value={debitSection} />
			<input type="hidden" name="credit_section" value={creditSection} />

			<Field.Group>
				<div class="grid gap-3 sm:grid-cols-3">
					<Field.Field>
						<Field.Label for="jrn-code">Code journal</Field.Label>
						<Input id="jrn-code" name="journal_code" bind:value={journalCode} required />
					</Field.Field>

					<Field.Field>
						<Field.Label for="jrn-lbl">Libellé journal</Field.Label>
						<Input id="jrn-lbl" name="journal_label" bind:value={journalLabel} required />
					</Field.Field>

					<Field.Field>
						<Field.Label for="jrn-ref">Réf. pièce</Field.Label>
						<Input id="jrn-ref" name="document_ref" bind:value={documentRef} required />
					</Field.Field>
				</div>

				<Field.Field>
					<Field.Label for="jrn-entry-lbl">Libellé de l’écriture</Field.Label>
					<Input id="jrn-entry-lbl" name="entry_label" bind:value={entryLabel} required />
				</Field.Field>

				<div class="flex flex-col gap-2.5 rounded-md border p-3">
					<span class="text-xs font-semibold uppercase">Ligne au Débit</span>
					<div class="grid gap-2 sm:grid-cols-3">
						<Input
							name="debit_account"
							placeholder="Compte (681100)"
							class="font-mono"
							bind:value={debitAccount}
							required
						/>
						<Input
							name="debit_account_label"
							placeholder="Libellé compte"
							bind:value={debitAccountLabel}
							required
						/>
						<Input
							name="debit_euro"
							placeholder="Montant €"
							inputmode="decimal"
							bind:value={debitEuro}
							required
						/>
					</div>
					<Select.Root
						type="single"
						value={debitSection}
						onValueChange={(next) => {
							if (next) debitSection = next;
						}}
					>
						<Select.Trigger class="w-full">
							{formatStatementSectionLabel(debitSection)}
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
				</div>

				<div class="flex flex-col gap-2.5 rounded-md border p-3">
					<span class="text-xs font-semibold uppercase">Ligne au Crédit</span>
					<div class="grid gap-2 sm:grid-cols-3">
						<Input
							name="credit_account"
							placeholder="Compte (281300)"
							class="font-mono"
							bind:value={creditAccount}
							required
						/>
						<Input
							name="credit_account_label"
							placeholder="Libellé compte"
							bind:value={creditAccountLabel}
							required
						/>
						<Input
							name="credit_euro"
							placeholder="Montant €"
							inputmode="decimal"
							bind:value={creditEuro}
							required
						/>
					</div>
					<Select.Root
						type="single"
						value={creditSection}
						onValueChange={(next) => {
							if (next) creditSection = next;
						}}
					>
						<Select.Trigger class="w-full">
							{formatStatementSectionLabel(creditSection)}
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
				</div>
			</Field.Group>

			<div class="flex items-center justify-between text-xs">
				<span class="tabular-nums">
					Débit : <strong>{formatCentsToEuro(journalBalanceCheck.debitCents)}</strong> · Crédit :
					<strong>{formatCentsToEuro(journalBalanceCheck.creditCents)}</strong>
				</span>
				<Badge variant={journalBalanceCheck.balanced ? 'secondary' : 'destructive'}>
					{journalBalanceCheck.balanced ? 'Équilibrée' : 'Déséquilibrée'}
				</Badge>
			</div>

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
				<Button
					type="submit"
					disabled={isSubmitting || !activeEntityId || !journalBalanceCheck.balanced}
				>
					{#if isSubmitting}
						<Spinner data-icon="inline-start" />
					{/if}
					Valider l’écriture au journal
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
