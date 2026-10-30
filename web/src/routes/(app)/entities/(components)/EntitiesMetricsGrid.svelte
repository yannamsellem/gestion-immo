<script lang="ts">
	import * as Card from '#lib/components/ui/card/index.js';
	import { formatCentsToEuro } from '#lib/core/formatters/money';

	let {
		totalAcquisitionCents,
		totalWorkStockCents,
		totalBookValueCents,
		totalLotsCount,
		entitiesCount
	}: {
		totalAcquisitionCents: number;
		totalWorkStockCents: number;
		totalBookValueCents: number;
		totalLotsCount: number;
		entitiesCount: number;
	} = $props();
</script>

<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
	<Card.Root>
		<Card.Header class="pb-2">
			<Card.Description>Valeur d’acquisition brute</Card.Description>
			<Card.Title class="text-xl font-semibold tabular-nums">
				{formatCentsToEuro(totalAcquisitionCents)}
			</Card.Title>
		</Card.Header>
		<Card.Content>
			<p class="text-xs text-muted-foreground">Actifs immobilisés & stocks</p>
		</Card.Content>
	</Card.Root>

	<Card.Root>
		<Card.Header class="pb-2">
			<Card.Description>En-cours & stocks travaux</Card.Description>
			<Card.Title class="text-xl font-semibold tabular-nums">
				{formatCentsToEuro(totalWorkStockCents)}
			</Card.Title>
		</Card.Header>
		<Card.Content>
			<p class="text-xs text-muted-foreground">Travaux capitalisés rattachés</p>
		</Card.Content>
	</Card.Root>

	<Card.Root>
		<Card.Header class="pb-2">
			<Card.Description>Valeur nette comptable (VNC)</Card.Description>
			<Card.Title class="text-xl font-semibold tabular-nums">
				{formatCentsToEuro(totalBookValueCents)}
			</Card.Title>
		</Card.Header>
		<Card.Content>
			<p class="text-xs text-muted-foreground">Acquisition + Stock travaux</p>
		</Card.Content>
	</Card.Root>

	<Card.Root>
		<Card.Header class="pb-2">
			<Card.Description>Lots en portefeuille</Card.Description>
			<Card.Title class="text-xl font-semibold tabular-nums">{totalLotsCount}</Card.Title>
		</Card.Header>
		<Card.Content>
			<p class="text-xs text-muted-foreground">Sur {entitiesCount} entité(s)</p>
		</Card.Content>
	</Card.Root>
</div>
