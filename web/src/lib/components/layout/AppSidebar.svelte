<script lang="ts">
	import { enhance } from '$app/forms';
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
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Sidebar from '#lib/components/ui/sidebar/index.js';
	import type { UserDTO } from '#lib/modules/auth/types';
	import type { LegalEntityDTO } from '#lib/modules/portfolio/types';
	import EntityWorkspaceSwitcher from './EntityWorkspaceSwitcher.svelte';

	let {
		user = null,
		entities = [],
		pendingRentTasksCount = 0,
		openIncidentsCount = 0
	}: {
		user?: UserDTO | null;
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
</script>

<Sidebar.Root collapsible="offcanvas">
	<Sidebar.Header class="gap-3 border-b p-3">
		<div class="flex flex-col px-1">
			<span class="text-sm font-semibold tracking-tight">Gestion Immo</span>
			<span class="text-[11px] text-muted-foreground">Back-office Patrimonial</span>
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
		{#if user}
			<div class="flex items-center justify-between gap-2">
				<div class="flex min-w-0 flex-col">
					<span class="truncate text-xs font-medium">{user.full_name}</span>
					<span class="truncate text-[11px] text-muted-foreground">
						{user.email} · {user.role}
					</span>
				</div>
				<form method="POST" action={resolve('/(auth)/logout')} use:enhance>
					<Button type="submit" variant="ghost" size="icon-sm" aria-label="Se déconnecter">
						<IconLogout />
					</Button>
				</form>
			</div>
		{/if}
	</Sidebar.Footer>
	<Sidebar.Rail />
</Sidebar.Root>
