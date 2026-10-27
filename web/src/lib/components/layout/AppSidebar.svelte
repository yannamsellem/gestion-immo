<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import IconBuildingBank from '@tabler/icons-svelte/icons/building-bank';
	import IconCalculator from '@tabler/icons-svelte/icons/calculator';
	import IconClipboardCheck from '@tabler/icons-svelte/icons/clipboard-check';
	import IconFileText from '@tabler/icons-svelte/icons/file-text';
	import IconHome from '@tabler/icons-svelte/icons/home';
	import IconLayoutDashboard from '@tabler/icons-svelte/icons/layout-dashboard';
	import IconLogout from '@tabler/icons-svelte/icons/logout';
	import IconReceiptEuro from '@tabler/icons-svelte/icons/receipt-euro';
	import IconShieldLock from '@tabler/icons-svelte/icons/shield-lock';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Sidebar from '#lib/components/ui/sidebar/index.js';
	import { authStore } from '#lib/modules/auth/auth-store.svelte';
	import type { LegalEntityDTO } from '#lib/modules/portfolio/types';
	import { apiRegistry } from '#lib/modules/registry';
	import EntityWorkspaceSwitcher from './EntityWorkspaceSwitcher.svelte';

	let {
		entities = [],
		pendingRentTasksCount = 0,
		openIncidentsCount = 0
	}: {
		entities?: LegalEntityDTO[];
		pendingRentTasksCount?: number;
		openIncidentsCount?: number;
	} = $props();

	const navItems = $derived([
		{
			route: '/(app)' as const,
			label: 'Cockpit Opérationnel',
			icon: IconLayoutDashboard,
			badge: 0
		},
		{
			route: '/(app)/rents' as const,
			label: 'Loyers & Relances',
			icon: IconReceiptEuro,
			badge: pendingRentTasksCount
		},
		{
			route: '/(app)/inspections' as const,
			label: 'Incidents & États des lieux',
			icon: IconClipboardCheck,
			badge: openIncidentsCount
		},
		{
			route: '/(app)/properties' as const,
			label: 'Biens & Lots',
			icon: IconHome,
			badge: 0
		},
		{
			route: '/(app)/leases' as const,
			label: 'Baux & Locataires',
			icon: IconFileText,
			badge: 0
		},
		{
			route: '/(app)/accounting' as const,
			label: 'Comptabilité & Fiscalité',
			icon: IconCalculator,
			badge: 0
		},
		{
			route: '/(app)/entities' as const,
			label: 'Entités & Associés',
			icon: IconBuildingBank,
			badge: 0
		},
		{
			route: '/(app)/admin' as const,
			label: 'Administration & Accès',
			icon: IconShieldLock,
			badge: 0
		}
	]);

	function isRouteActive(href: string, currentPath: string): boolean {
		if (href === '/') {
			return currentPath === '/';
		}
		return currentPath === href || currentPath.startsWith(`${href}/`);
	}

	async function handleLogout(): Promise<void> {
		const refreshToken = authStore.refreshToken;
		if (refreshToken) {
			try {
				await apiRegistry.auth.logout(refreshToken);
			} catch {
				// Proceed with local session cleanup even if backend token already expired
			}
		}
		authStore.clearSession();
		await goto(resolve('/(auth)/login'));
	}
</script>

<Sidebar.Root collapsible="offcanvas">
	<Sidebar.Header class="gap-3 border-b p-3">
		<div class="flex items-center justify-between px-1">
			<div class="flex flex-col">
				<span class="text-sm font-semibold tracking-tight">Gestion Immo</span>
				<span class="text-[11px] text-muted-foreground">Back-office Patrimonial</span>
			</div>
			<Badge variant="secondary">Svelte 5</Badge>
		</div>
		<EntityWorkspaceSwitcher {entities} />
	</Sidebar.Header>

	<Sidebar.Content>
		<Sidebar.Group>
			<Sidebar.GroupLabel>Pilotage Métier</Sidebar.GroupLabel>
			<Sidebar.GroupContent>
				<Sidebar.Menu>
					{#each navItems as item (item.route)}
						{@const Icon = item.icon}
						{@const resolvedHref = resolve(item.route)}
						<Sidebar.MenuItem>
							<Sidebar.MenuButton isActive={isRouteActive(resolvedHref, page.url.pathname)}>
								{#snippet child({ props })}
									<a href={resolvedHref} {...props}>
										<Icon />
										<span>{item.label}</span>
									</a>
								{/snippet}
							</Sidebar.MenuButton>
							{#if item.badge > 0}
								<Sidebar.MenuBadge>{item.badge}</Sidebar.MenuBadge>
							{/if}
						</Sidebar.MenuItem>
					{/each}
				</Sidebar.Menu>
			</Sidebar.GroupContent>
		</Sidebar.Group>
	</Sidebar.Content>

	<Sidebar.Footer class="border-t p-3">
		{#if authStore.user}
			<div class="flex items-center justify-between gap-2">
				<div class="flex min-w-0 flex-col">
					<span class="truncate text-xs font-medium">{authStore.user.full_name}</span>
					<span class="truncate text-[11px] text-muted-foreground">
						{authStore.user.email} · {authStore.user.role}
					</span>
				</div>
				<Button variant="ghost" size="icon-sm" onclick={handleLogout} aria-label="Se déconnecter">
					<IconLogout />
				</Button>
			</div>
		{/if}
	</Sidebar.Footer>
	<Sidebar.Rail />
</Sidebar.Root>
