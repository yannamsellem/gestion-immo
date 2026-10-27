<script lang="ts">
	import IconBuildingSkyscraper from '@tabler/icons-svelte/icons/building-skyscraper';
	import * as Select from '#lib/components/ui/select/index.js';
	import { workspaceStore } from '#lib/core/workspace/workspace.svelte';
	import type { LegalEntityDTO } from '#lib/modules/portfolio/types';

	let {
		entities = []
	}: {
		entities?: LegalEntityDTO[];
	} = $props();

	const triggerLabel = $derived.by(() => {
		if (workspaceStore.selectedEntityId === 'ALL') {
			return `Toutes les entités (${entities.length})`;
		}
		const found = entities.find((e) => e.id === workspaceStore.selectedEntityId);
		return found ? `${found.name} (${found.entity_type})` : 'Toutes les entités';
	});
</script>

<div class="flex flex-col gap-1">
	<span class="px-1 text-[10px] font-medium tracking-wider text-muted-foreground uppercase">
		Contexte juridique
	</span>
	<Select.Root
		type="single"
		value={workspaceStore.selectedEntityId}
		onValueChange={(value) => {
			if (value) {
				workspaceStore.setEntityId(value);
			}
		}}
	>
		<Select.Trigger class="w-full justify-between" aria-label="Sélecteur d'entité juridique">
			<span class="flex items-center gap-1.5 truncate">
				<IconBuildingSkyscraper class="shrink-0 text-muted-foreground" />
				<span class="truncate">{triggerLabel}</span>
			</span>
		</Select.Trigger>
		<Select.Content>
			<Select.Group>
				<Select.Label>Périmètre actif</Select.Label>
				<Select.Item value="ALL" label="Toutes les entités">
					Toutes les entités ({entities.length})
				</Select.Item>
				{#each entities as entity (entity.id)}
					<Select.Item value={entity.id} label={entity.name}>
						{entity.name} · {entity.entity_type}
					</Select.Item>
				{/each}
			</Select.Group>
		</Select.Content>
	</Select.Root>
</div>
