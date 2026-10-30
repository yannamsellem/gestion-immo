<script lang="ts">
	import * as Card from '#lib/components/ui/card/index.js';
	import { formatPeriodMonthFr } from '#lib/core/formatters/date';
	import { formatCentsToEuro } from '#lib/core/formatters/money';

	let {
		periodMonth,
		totalCollectedCents,
		totalExpectedCents,
		totalRemainingDueCents,
		totalOverdueLeasesCount,
		totalOpenIncidentsCount
	}: {
		periodMonth: string;
		totalCollectedCents: number;
		totalExpectedCents: number;
		totalRemainingDueCents: number;
		totalOverdueLeasesCount: number;
		totalOpenIncidentsCount: number;
	} = $props();
</script>

<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
	<Card.Root>
		<Card.Header class="pb-2">
			<Card.Description>
				Encaissé ({formatPeriodMonthFr(periodMonth)})
			</Card.Description>
			<Card.Title class="text-xl font-semibold tabular-nums">
				{formatCentsToEuro(totalCollectedCents)}
			</Card.Title>
		</Card.Header>
		<Card.Content>
			<p class="text-xs text-muted-foreground tabular-nums">
				Sur {formatCentsToEuro(totalExpectedCents)} attendus
			</p>
		</Card.Content>
	</Card.Root>

	<Card.Root>
		<Card.Header class="pb-2">
			<Card.Description>Reste à encaisser</Card.Description>
			<Card.Title class="text-xl font-semibold tabular-nums">
				{formatCentsToEuro(totalRemainingDueCents)}
			</Card.Title>
		</Card.Header>
		<Card.Content>
			<p class="text-xs text-muted-foreground">Échéances en attente de règlement</p>
		</Card.Content>
	</Card.Root>

	<Card.Root>
		<Card.Header class="pb-2">
			<Card.Description>Impayés à relancer</Card.Description>
			<Card.Title class="text-xl font-semibold tabular-nums">
				{totalOverdueLeasesCount}
			</Card.Title>
		</Card.Header>
		<Card.Content>
			<p class="text-xs text-muted-foreground">Baux en position débitrice</p>
		</Card.Content>
	</Card.Root>

	<Card.Root>
		<Card.Header class="pb-2">
			<Card.Description>Incidents & travaux ouverts</Card.Description>
			<Card.Title class="text-xl font-semibold tabular-nums">
				{totalOpenIncidentsCount}
			</Card.Title>
		</Card.Header>
		<Card.Content>
			<p class="text-xs text-muted-foreground">Sinistres ou interventions en cours</p>
		</Card.Content>
	</Card.Root>
</div>
