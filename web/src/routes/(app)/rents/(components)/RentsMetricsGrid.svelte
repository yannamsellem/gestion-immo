<script lang="ts">
	import * as Card from '#lib/components/ui/card/index.js';
	import { formatCentsToEuro } from '#lib/core/formatters/money';

	let {
		expectedTotalCents,
		collectedTotalCents,
		remainingTotalCents,
		overdueCount
	}: {
		expectedTotalCents: number;
		collectedTotalCents: number;
		remainingTotalCents: number;
		overdueCount: number;
	} = $props();
</script>

<div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
	<Card.Root>
		<Card.Header class="pb-2">
			<Card.Description>Appelé du mois (TTC / CC)</Card.Description>
			<Card.Title class="font-mono text-xl tabular-nums">
				{formatCentsToEuro(expectedTotalCents)}
			</Card.Title>
		</Card.Header>
	</Card.Root>

	<Card.Root>
		<Card.Header class="pb-2">
			<Card.Description>Encaissé sur l’échéance</Card.Description>
			<Card.Title class="font-mono text-xl tabular-nums">
				{formatCentsToEuro(collectedTotalCents)}
			</Card.Title>
		</Card.Header>
	</Card.Root>

	<Card.Root>
		<Card.Header class="pb-2">
			<Card.Description>Reste à encaisser</Card.Description>
			<Card.Title
				class="font-mono text-xl tabular-nums {remainingTotalCents > 0 ? 'text-destructive' : ''}"
			>
				{formatCentsToEuro(remainingTotalCents)}
			</Card.Title>
		</Card.Header>
	</Card.Root>

	<Card.Root>
		<Card.Header class="pb-2">
			<Card.Description>Baux en impayé (arriéré)</Card.Description>
			<Card.Title
				class="font-mono text-xl tabular-nums {overdueCount > 0 ? 'text-destructive' : ''}"
			>
				{overdueCount}
			</Card.Title>
		</Card.Header>
	</Card.Root>
</div>
