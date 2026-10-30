<script lang="ts">
	import IconBuildingBank from '@tabler/icons-svelte/icons/building-bank';
	import IconCash from '@tabler/icons-svelte/icons/cash';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Empty from '#lib/components/ui/empty/index.js';
	import * as Table from '#lib/components/ui/table/index.js';
	import { formatIsoDateFr } from '#lib/core/formatters/date';
	import { formatCentsToEuro } from '#lib/core/formatters/money';
	import type { CashJournalEntryDTO } from '#lib/modules/payment/types';
	import type { LegalEntityDTO } from '#lib/modules/portfolio/types';

	let {
		cashEntries,
		entities,
		onRecordDeposit
	}: {
		cashEntries: CashJournalEntryDTO[];
		entities: LegalEntityDTO[];
		onRecordDeposit: () => void;
	} = $props();

	const entityMap = $derived(new Map(entities.map((entity) => [entity.id, entity])));
	const totalDebitCents = $derived(cashEntries.reduce((sum, entry) => sum + entry.debit_cents, 0));
	const totalCreditCents = $derived(
		cashEntries.reduce((sum, entry) => sum + entry.credit_cents, 0)
	);
	const netCashBalanceCents = $derived(totalDebitCents - totalCreditCents);
</script>

<div class="grid gap-3 sm:grid-cols-3">
	<Card.Root>
		<Card.Header class="pb-2">
			<Card.Description>Encaissements espèces (Débit 530)</Card.Description>
			<Card.Title class="font-mono text-xl tabular-nums">
				{formatCentsToEuro(totalDebitCents)}
			</Card.Title>
		</Card.Header>
	</Card.Root>

	<Card.Root>
		<Card.Header class="pb-2">
			<Card.Description>Remises en banque (Crédit 530)</Card.Description>
			<Card.Title class="font-mono text-xl tabular-nums">
				{formatCentsToEuro(totalCreditCents)}
			</Card.Title>
		</Card.Header>
	</Card.Root>

	<Card.Root>
		<Card.Header class="pb-2">
			<Card.Description>Solde de caisse théorique</Card.Description>
			<Card.Title class="font-mono text-xl tabular-nums">
				{formatCentsToEuro(netCashBalanceCents)}
			</Card.Title>
		</Card.Header>
	</Card.Root>
</div>

{#if cashEntries.length === 0}
	<Card.Root>
		<Card.Content class="py-10">
			<Empty.Root>
				<Empty.Header>
					<Empty.Media variant="icon">
						<IconCash class="size-5" />
					</Empty.Media>
					<Empty.Title>Aucune opération en espèces enregistrée</Empty.Title>
					<Empty.Description>
						Les règlements de loyers en espèces (411 → 530) et vos remises d’espèces en banque (530
						→ 512) apparaîtront chronologiquement ici.
					</Empty.Description>
				</Empty.Header>
				<Empty.Content>
					<Button variant="outline" onclick={onRecordDeposit}>
						<IconBuildingBank data-icon="inline-start" />
						Remise espèces (530 → 512)
					</Button>
				</Empty.Content>
			</Empty.Root>
		</Card.Content>
	</Card.Root>
{:else}
	<Card.Root>
		<Card.Header
			class="flex flex-col justify-between gap-2 border-b pb-3 sm:flex-row sm:items-center"
		>
			<div>
				<Card.Title class="flex items-center gap-1.5 text-sm font-semibold">
					<IconCash class="size-4" />
					Journal chronologique PCG 530
				</Card.Title>
				<Card.Description>
					Traçabilité des reçus espèces et remises bancaires sur le périmètre sélectionné
				</Card.Description>
			</div>
			<Badge variant="outline">{cashEntries.length} écriture(s)</Badge>
		</Card.Header>
		<Card.Content class="p-0">
			<Table.Root>
				<Table.Header>
					<Table.Row>
						<Table.Head class="pl-4">Date</Table.Head>
						<Table.Head>Entité</Table.Head>
						<Table.Head>Type</Table.Head>
						<Table.Head>Libellé & Reçu</Table.Head>
						<Table.Head>Comptes PCG</Table.Head>
						<Table.Head class="text-right">Débit 530</Table.Head>
						<Table.Head class="pr-4 text-right">Crédit 530</Table.Head>
					</Table.Row>
				</Table.Header>
				<Table.Body>
					{#each cashEntries as entry (entry.id)}
						<Table.Row>
							<Table.Cell class="pl-4 text-xs">{formatIsoDateFr(entry.entry_date)}</Table.Cell>
							<Table.Cell class="text-xs text-muted-foreground">
								{entityMap.get(entry.legal_entity_id)?.name ?? 'Entité'}
							</Table.Cell>
							<Table.Cell>
								<Badge variant={entry.entry_type === 'CASH_RECEIPT' ? 'secondary' : 'outline'}>
									{entry.entry_type === 'CASH_RECEIPT' ? 'Reçu espèces' : 'Remise banque'}
								</Badge>
							</Table.Cell>
							<Table.Cell class="text-xs">
								<span class="font-medium">{entry.label}</span>
								{#if entry.receipt_number}
									<span class="ml-1 font-mono text-muted-foreground">
										({entry.receipt_number})
									</span>
								{/if}
							</Table.Cell>
							<Table.Cell class="font-mono text-xs">
								{entry.account_code} / {entry.counterpart_account_code}
							</Table.Cell>
							<Table.Cell class="text-right font-mono tabular-nums">
								{entry.debit_cents > 0 ? formatCentsToEuro(entry.debit_cents) : '—'}
							</Table.Cell>
							<Table.Cell class="pr-4 text-right font-mono tabular-nums">
								{entry.credit_cents > 0 ? formatCentsToEuro(entry.credit_cents) : '—'}
							</Table.Cell>
						</Table.Row>
					{/each}
				</Table.Body>
			</Table.Root>
		</Card.Content>
	</Card.Root>
{/if}
