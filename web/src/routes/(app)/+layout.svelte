<script lang="ts">
	import { untrack } from 'svelte';
	import AppSidebar from '#lib/components/layout/AppSidebar.svelte';
	import ThemeToggle from '#lib/components/layout/ThemeToggle.svelte';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Separator } from '#lib/components/ui/separator/index.js';
	import * as Sidebar from '#lib/components/ui/sidebar/index.js';
	import { formatPeriodMonthFr } from '#lib/core/formatters/date';
	import { workspaceStore } from '#lib/core/workspace/workspace.svelte';
	import type { LayoutProps } from './$types';

	let { data, children }: LayoutProps = $props();

	$effect(() => {
		const entities = data.entities;
		const selectedEntityId = data.selectedEntityId;
		const selectedPeriodMonth = data.selectedPeriodMonth;
		untrack(() => {
			workspaceStore.hydrateFromServer({
				entities,
				selectedEntityId,
				selectedPeriodMonth
			});
		});
	});
</script>

<Sidebar.Provider>
	<AppSidebar
		user={data.user}
		entities={data.entities}
		openIncidentsCount={data.openIncidentsCount}
	/>
	<Sidebar.Inset>
		<header
			class="sticky top-0 flex h-12 shrink-0 items-center justify-between gap-2 border-b bg-background/95 px-4 backdrop-blur"
		>
			<div class="flex items-center gap-2">
				<Sidebar.Trigger />
				<Separator orientation="vertical" class="h-4" />
				<Badge variant="outline">
					{#if workspaceStore.activeEntity}
						{workspaceStore.activeEntity.name}
					{:else}
						Toutes les entités ({data.entities.length})
					{/if}
				</Badge>
				<span class="hidden text-xs text-muted-foreground sm:inline">
					· Période : {formatPeriodMonthFr(workspaceStore.selectedPeriodMonth)}
				</span>
			</div>
			<div class="flex items-center gap-2">
				<ThemeToggle />
			</div>
		</header>

		<main class="flex flex-1 flex-col gap-6 p-4 md:p-6">
			{@render children()}
		</main>
	</Sidebar.Inset>
</Sidebar.Provider>
