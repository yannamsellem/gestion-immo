<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import IconAlertTriangle from '@tabler/icons-svelte/icons/alert-triangle';
	import IconFileExport from '@tabler/icons-svelte/icons/file-export';
	import { toast } from 'svelte-sonner';
	import AccountingContextToolbar from '#lib/components/accounting/AccountingContextToolbar.svelte';
	import PageHeader from '#lib/components/layout/PageHeader.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { toggleDisabledLineId } from '#lib/modules/accounting/statement-state';
	import type { ComputedFinancialStatementsDTO } from '#lib/modules/accounting/types';
	import FinancialStatementsCard from './(components)/FinancialStatementsCard.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	let disabledLineIds = $state<string[]>([]);
	let localStatements = $state<ComputedFinancialStatementsDTO | null>(null);
	let isRecomputing = $state(false);
	let recomputeFormEl = $state<HTMLFormElement | null>(null);

	$effect(() => {
		localStatements = data.statements;
		disabledLineIds = data.disabledLineIds;
	});

	const statements = $derived(localStatements ?? data.statements);

	function handleToggleStatementLine(lineId: string): void {
		disabledLineIds = toggleDisabledLineId(disabledLineIds, lineId);
		queueMicrotask(() => {
			recomputeFormEl?.requestSubmit();
		});
	}
</script>

<form
	bind:this={recomputeFormEl}
	method="POST"
	action="?/recompute"
	use:enhance={() => {
		isRecomputing = true;
		return async ({ result }) => {
			isRecomputing = false;
			if (result.type === 'success') {
				const payload = result.data as
					| {
							recomputedStatements?: ComputedFinancialStatementsDTO;
							disabledLineIds?: string[];
					  }
					| undefined;
				if (payload?.recomputedStatements) {
					localStatements = payload.recomputedStatements;
					disabledLineIds = payload.disabledLineIds ?? [];
				}
				return;
			}
			if (result.type === 'failure') {
				const payload = result.data as { actionError?: string } | undefined;
				toast.error(payload?.actionError ?? 'Erreur lors du recalcul interactif.');
				return;
			}
			if (result.type === 'error') {
				toast.error(result.error?.message ?? 'Erreur serveur lors du recalcul interactif.');
			}
		};
	}}
	class="hidden"
>
	<input type="hidden" name="entity_id" value={data.activeEntityId} />
	<input type="hidden" name="fiscal_year" value={data.fiscalYear} />
	<input type="hidden" name="disabled_line_ids_json" value={JSON.stringify(disabledLineIds)} />
</form>

<PageHeader
	title="Bilan & Compte de résultat"
	subtitle="Simulation interactive en temps réel (activation ou neutralisation de postes type amortissements) et soldes intermédiaires de gestion"
>
	{#snippet actions()}
		<AccountingContextToolbar
			entities={data.entities}
			activeEntityId={data.activeEntityId}
			fiscalYear={data.fiscalYear}
		/>

		<Button
			size="sm"
			variant="outline"
			href={`${resolve('/(app)/accounting/tax')}?year=${data.fiscalYear}`}
		>
			<IconFileExport data-icon="inline-start" />
			Liasse & Export FEC →
		</Button>
	{/snippet}
</PageHeader>

{#if data.errorMessage}
	<Alert.Root variant="destructive">
		<IconAlertTriangle />
		<Alert.Title>Erreur</Alert.Title>
		<Alert.Description>{data.errorMessage}</Alert.Description>
	</Alert.Root>
{/if}

<FinancialStatementsCard
	{statements}
	fiscalYear={data.fiscalYear}
	{disabledLineIds}
	{isRecomputing}
	onToggleLine={handleToggleStatementLine}
/>
