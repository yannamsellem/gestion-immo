<script lang="ts">
	import IconBuildingSkyscraper from '@tabler/icons-svelte/icons/building-skyscraper';
	import IconCheck from '@tabler/icons-svelte/icons/check';
	import IconLayoutGrid from '@tabler/icons-svelte/icons/layout-grid';
	import IconPlus from '@tabler/icons-svelte/icons/plus';
	import IconSelector from '@tabler/icons-svelte/icons/selector';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.js';
	import * as Sidebar from '#lib/components/ui/sidebar/index.js';
	import { useEntityDialogContext } from '#lib/core/workspace/entity-dialog-context';
	import { workspaceStore } from '#lib/core/workspace/workspace.svelte';
	import { formatEntityTypeLabel } from '#lib/modules/portfolio/labels';
	import type { LegalEntityDTO } from '#lib/modules/portfolio/types';

	let {
		entities
	}: {
		entities: LegalEntityDTO[];
	} = $props();

	const sidebar = Sidebar.useSidebar();
	const entityDialog = useEntityDialogContext();

	const selectedEntity = $derived(
		workspaceStore.selectedEntityId === 'ALL'
			? null
			: (entities.find((entity) => entity.id === workspaceStore.selectedEntityId) ?? null)
	);

	const activeTitle = $derived(selectedEntity ? selectedEntity.name : 'Toutes les entités');
	const activeSubtitle = $derived(
		selectedEntity
			? formatEntityTypeLabel(selectedEntity.entity_type)
			: `${entities.length} structure${entities.length > 1 ? 's' : ''} juridique${entities.length > 1 ? 's' : ''}`
	);
</script>

<Sidebar.Menu>
	<Sidebar.MenuItem>
		<DropdownMenu.Root>
			<DropdownMenu.Trigger>
				{#snippet child({ props })}
					<Sidebar.MenuButton
						{...props}
						size="lg"
						class="gap-3 text-sm data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
					>
						<div
							class="flex aspect-square size-8 shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground"
						>
							<IconBuildingSkyscraper class="size-5" />
						</div>
						<div class="grid flex-1 text-left text-sm leading-tight">
							<span class="truncate font-semibold">{activeTitle}</span>
							<span class="truncate text-xs text-muted-foreground">{activeSubtitle}</span>
						</div>
						<IconSelector class="ml-auto size-4" />
					</Sidebar.MenuButton>
				{/snippet}
			</DropdownMenu.Trigger>

			<DropdownMenu.Content
				class="w-(--bits-dropdown-menu-anchor-width) min-w-60 rounded-lg"
				align="start"
				side={sidebar.isMobile ? 'bottom' : 'right'}
				sideOffset={4}
			>
				<DropdownMenu.Group>
					<DropdownMenu.Label class="text-xs text-muted-foreground">
						Entités juridiques
					</DropdownMenu.Label>

					{#if entities.length > 1}
						<DropdownMenu.Item
							class="cursor-pointer gap-2.5 p-2 text-sm"
							onclick={() => {
								workspaceStore.setEntityId('ALL');
							}}
						>
							<div class="flex size-6 items-center justify-center rounded-md border">
								<IconLayoutGrid class="size-4 shrink-0" />
							</div>
							<span class="flex-1 truncate font-medium">
								Toutes les entités ({entities.length})
							</span>
							{#if workspaceStore.selectedEntityId === 'ALL'}
								<IconCheck class="size-4 text-primary" />
							{/if}
						</DropdownMenu.Item>
						<DropdownMenu.Separator />
					{/if}

					{#each entities as entity, index (entity.id)}
						<DropdownMenu.Item
							class="cursor-pointer gap-2.5 p-2 text-sm"
							onclick={() => {
								workspaceStore.setEntityId(entity.id);
							}}
						>
							<div class="flex size-6 items-center justify-center rounded-md border">
								<IconBuildingSkyscraper class="size-4 shrink-0" />
							</div>
							<div class="flex min-w-0 flex-1 flex-col">
								<span class="truncate font-medium">{entity.name}</span>
								<span class="truncate text-[11px] text-muted-foreground">
									{formatEntityTypeLabel(entity.entity_type)}
								</span>
							</div>
							{#if workspaceStore.selectedEntityId === entity.id}
								<IconCheck class="size-4 text-primary" />
							{:else if index < 9}
								<DropdownMenu.Shortcut>⌘{index + 1}</DropdownMenu.Shortcut>
							{/if}
						</DropdownMenu.Item>
					{/each}
				</DropdownMenu.Group>

				<DropdownMenu.Separator />

				<DropdownMenu.Item
					class="cursor-pointer gap-2.5 p-2 text-sm"
					onclick={() => {
						entityDialog.openCreateDialog();
					}}
				>
					<div class="flex size-6 items-center justify-center rounded-md border bg-transparent">
						<IconPlus class="size-4" />
					</div>
					<span class="font-medium text-muted-foreground">Créer une entité juridique…</span>
				</DropdownMenu.Item>
			</DropdownMenu.Content>
		</DropdownMenu.Root>
	</Sidebar.MenuItem>
</Sidebar.Menu>
