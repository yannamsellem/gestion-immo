<script lang="ts">
	import { Input } from '#lib/components/ui/input/index.js';
	import * as Select from '#lib/components/ui/select/index.js';
	import { workspaceStore } from '#lib/core/workspace/workspace.svelte';
	import type { LegalEntityDTO } from '#lib/modules/portfolio/types';

	let {
		entities,
		activeEntityId,
		fiscalYear
	}: {
		entities: LegalEntityDTO[];
		activeEntityId: string;
		fiscalYear: number;
	} = $props();

	const activeEntity = $derived(entities.find((entity) => entity.id === activeEntityId) ?? null);
</script>

{#if entities.length > 0}
	<Select.Root
		type="single"
		value={activeEntityId}
		onValueChange={(next) => {
			if (next) {
				workspaceStore.setEntityId(next);
			}
		}}
	>
		<Select.Trigger class="h-8 w-48">
			{activeEntity?.name ?? 'Sélectionner une entité'}
		</Select.Trigger>
		<Select.Content>
			<Select.Group>
				{#each entities as entity (entity.id)}
					<Select.Item value={entity.id} label={entity.name}>{entity.name}</Select.Item>
				{/each}
			</Select.Group>
		</Select.Content>
	</Select.Root>
{/if}

<form method="GET">
	<Input
		type="number"
		name="year"
		min="2000"
		max="2100"
		value={String(fiscalYear)}
		onchange={(event) => {
			const input = event.currentTarget as HTMLInputElement;
			const parsed = Number.parseInt(input.value, 10);
			if (!Number.isNaN(parsed) && parsed >= 2000 && parsed <= 2100) {
				input.form?.requestSubmit();
			}
		}}
		class="h-8 w-24 font-mono"
		aria-label="Exercice fiscal"
	/>
</form>
