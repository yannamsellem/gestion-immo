<script lang="ts">
	import IconBook from '@tabler/icons-svelte/icons/book';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Empty from '#lib/components/ui/empty/index.js';
	import * as Table from '#lib/components/ui/table/index.js';
	import { formatIsoDateFr } from '#lib/core/formatters/date';
	import { formatCentsToEuro } from '#lib/core/formatters/money';
	import type { JournalEntryDTO } from '#lib/modules/accounting/types';

	let {
		journalEntries,
		fiscalYear,
		onCreateEntry
	}: {
		journalEntries: JournalEntryDTO[];
		fiscalYear: number;
		onCreateEntry: () => void;
	} = $props();
</script>

<Card.Root>
	<Card.Header
		class="flex flex-col justify-between gap-2 border-b pb-3 sm:flex-row sm:items-center"
	>
		<div>
			<Card.Title class="text-sm font-semibold">
				Livre-Journal PCG — Écritures validées {fiscalYear} ({journalEntries.length})
			</Card.Title>
			<Card.Description>
				Toutes les écritures en partie double validées ci-dessous sont incluses dans l’export
				normatif DGFiP FEC (18 colonnes).
			</Card.Description>
		</div>
		<Button size="sm" onclick={onCreateEntry}>
			<IconBook data-icon="inline-start" />
			Nouvelle écriture OD/PCG
		</Button>
	</Card.Header>

	<Card.Content class="p-0">
		{#if journalEntries.length === 0}
			<div class="py-10">
				<Empty.Root>
					<Empty.Header>
						<Empty.Media variant="icon">
							<IconBook class="size-5" />
						</Empty.Media>
						<Empty.Title>Aucune écriture comptable sur l’exercice {fiscalYear}</Empty.Title>
						<Empty.Description>
							Saisissez une écriture d’opérations diverses (amortissements, provisions, charges) en
							partie double équilibrée.
						</Empty.Description>
					</Empty.Header>
					<Empty.Content>
						<Button size="sm" onclick={onCreateEntry}>
							<IconBook data-icon="inline-start" />
							Saisir une écriture comptable
						</Button>
					</Empty.Content>
				</Empty.Root>
			</div>
		{:else}
			<Table.Root>
				<Table.Header>
					<Table.Row>
						<Table.Head class="pl-4">Date & N°</Table.Head>
						<Table.Head>Journal</Table.Head>
						<Table.Head>Pièce</Table.Head>
						<Table.Head>Libellé & Lignes PCG</Table.Head>
						<Table.Head class="text-right">Total Débit</Table.Head>
						<Table.Head class="pr-4 text-right">Total Crédit</Table.Head>
					</Table.Row>
				</Table.Header>
				<Table.Body>
					{#each journalEntries as entry (entry.id)}
						<Table.Row>
							<Table.Cell class="pl-4 text-xs">
								<span class="font-medium">{formatIsoDateFr(entry.entry_date)}</span>
								<span class="block font-mono text-[11px] text-muted-foreground">
									{entry.entry_number}
								</span>
							</Table.Cell>
							<Table.Cell>
								<Badge variant="outline">{entry.journal_code}</Badge>
							</Table.Cell>
							<Table.Cell class="font-mono text-xs">{entry.document_ref}</Table.Cell>
							<Table.Cell class="text-xs">
								<span class="font-semibold">{entry.label}</span>
								<div class="mt-1 flex flex-col gap-0.5 font-mono text-[11px] text-muted-foreground">
									{#each entry.lines as line (line.id)}
										<span>
											{line.account_code}
											{line.account_label} :
											{line.debit_cents > 0
												? `D ${formatCentsToEuro(line.debit_cents)}`
												: `C ${formatCentsToEuro(line.credit_cents)}`}
										</span>
									{/each}
								</div>
							</Table.Cell>
							<Table.Cell class="text-right font-mono tabular-nums">
								{formatCentsToEuro(entry.total_debit_cents)}
							</Table.Cell>
							<Table.Cell class="pr-4 text-right font-mono tabular-nums">
								{formatCentsToEuro(entry.total_credit_cents)}
							</Table.Cell>
						</Table.Row>
					{/each}
				</Table.Body>
			</Table.Root>
		{/if}
	</Card.Content>
</Card.Root>
