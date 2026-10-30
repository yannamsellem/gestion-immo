<script lang="ts">
	import { enhance } from '$app/forms';
	import { invalidateAll } from '$app/navigation';
	import { resolve } from '$app/paths';
	import IconAlertTriangle from '@tabler/icons-svelte/icons/alert-triangle';
	import IconLogout from '@tabler/icons-svelte/icons/logout';
	import IconRefresh from '@tabler/icons-svelte/icons/refresh';
	import { untrack } from 'svelte';
	import AppSidebar from '#lib/components/layout/AppSidebar.svelte';
	import ThemeToggle from '#lib/components/layout/ThemeToggle.svelte';
	import CreateEntityDialog from '#lib/components/portfolio/CreateEntityDialog.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Separator } from '#lib/components/ui/separator/index.js';
	import * as Sidebar from '#lib/components/ui/sidebar/index.js';
	import { formatPeriodMonthFr } from '#lib/core/formatters/date';
	import { setEntityDialogContext } from '#lib/core/workspace/entity-dialog-context';
	import { deriveWorkspaceGateState } from '#lib/core/workspace/onboarding-gate';
	import { workspaceStore } from '#lib/core/workspace/workspace.svelte';
	import type { LayoutProps } from './$types';

	let { data, children }: LayoutProps = $props();

	let createEntityDialogOpen = $state(false);
	let isRetryingLoad = $state(false);

	setEntityDialogContext({
		openCreateDialog: () => {
			createEntityDialogOpen = true;
		}
	});

	const gateState = $derived(
		deriveWorkspaceGateState({
			entitiesCount: data.entities.length,
			entitiesLoadError: data.entitiesLoadError
		})
	);

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

	async function handleRetryEntitiesLoad(): Promise<void> {
		isRetryingLoad = true;
		try {
			await invalidateAll();
		} finally {
			isRetryingLoad = false;
		}
	}
</script>

{#if gateState === 'load_error'}
	<div class="flex min-h-svh flex-col bg-background">
		<header class="flex h-12 items-center justify-between border-b px-4">
			<span class="text-sm font-semibold tracking-tight">Gestion Immo</span>
			<div class="flex items-center gap-2">
				<ThemeToggle />
				<form method="POST" action={resolve('/(auth)/logout')} use:enhance>
					<Button type="submit" variant="ghost" size="sm">
						<IconLogout data-icon="inline-start" />
						Se déconnecter
					</Button>
				</form>
			</div>
		</header>

		<main class="mx-auto flex w-full max-w-lg flex-1 flex-col justify-center gap-4 p-6">
			<Alert.Root variant="destructive">
				<IconAlertTriangle />
				<Alert.Title>Chargement du portefeuille indisponible</Alert.Title>
				<Alert.Description>
					{data.entitiesLoadError}
				</Alert.Description>
			</Alert.Root>
			<div class="flex items-center justify-end gap-2">
				<Button
					type="button"
					variant="outline"
					disabled={isRetryingLoad}
					onclick={handleRetryEntitiesLoad}
				>
					<IconRefresh data-icon="inline-start" />
					Réessayer
				</Button>
			</div>
		</main>
	</div>
{:else if gateState === 'onboarding_required'}
	<div class="flex min-h-svh flex-col bg-muted/30">
		<header
			class="pointer-events-auto relative z-60 flex h-12 items-center justify-between border-b bg-background/95 px-4 backdrop-blur"
		>
			<span class="text-sm font-semibold tracking-tight">Gestion Immo</span>
			<div class="flex items-center gap-2">
				<ThemeToggle />
				<form method="POST" action={resolve('/(auth)/logout')} use:enhance>
					<Button type="submit" variant="ghost" size="sm">
						<IconLogout data-icon="inline-start" />
						Se déconnecter
					</Button>
				</form>
			</div>
		</header>

		<CreateEntityDialog open={true} persistent={true} />
	</div>
{:else}
	<Sidebar.Provider>
		<AppSidebar user={data.user} entities={data.entities} />
		<Sidebar.Inset>
			<header
				class="sticky top-0 z-10 flex h-12 shrink-0 items-center justify-between gap-2 border-b bg-background/95 px-4 backdrop-blur"
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

	<CreateEntityDialog bind:open={createEntityDialogOpen} persistent={false} />
{/if}
