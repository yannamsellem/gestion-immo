<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import AppSidebar from '#lib/components/layout/AppSidebar.svelte';
	import ThemeToggle from '#lib/components/layout/ThemeToggle.svelte';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Separator } from '#lib/components/ui/separator/index.js';
	import * as Sidebar from '#lib/components/ui/sidebar/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import { formatPeriodMonthFr } from '#lib/core/formatters/date';
	import { workspaceStore } from '#lib/core/workspace/workspace.svelte';
	import { authStore } from '#lib/modules/auth/auth-store.svelte';
	import { apiRegistry } from '#lib/modules/registry';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();
	let openIncidentsCount = $state(0);

	$effect(() => {
		if (!authStore.initialized) {
			return;
		}
		if (!authStore.isAuthenticated) {
			void goto(resolve('/(auth)/login'));
			return;
		}

		workspaceStore.loadingEntities = true;
		void Promise.all([
			apiRegistry.portfolio.listEntities(),
			apiRegistry.inspection.listIncidents().catch(() => [])
		])
			.then(([entities, incidents]) => {
				workspaceStore.setEntities(entities);
				openIncidentsCount = incidents.filter(
					(inc) => inc.status === 'OPEN' || inc.status === 'IN_PROGRESS'
				).length;
			})
			.catch(() => {
				// Handled per-page or via 401 handler
			})
			.finally(() => {
				workspaceStore.loadingEntities = false;
			});
	});
</script>

{#if !authStore.initialized}
	<div class="flex min-h-screen items-center justify-center p-6">
		<Skeleton class="h-8 w-48" />
	</div>
{:else if authStore.isAuthenticated}
	<Sidebar.Provider>
		<AppSidebar entities={workspaceStore.entities} {openIncidentsCount} />
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
							Toutes les entités ({workspaceStore.entities.length})
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
{/if}
