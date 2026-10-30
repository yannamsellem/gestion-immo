<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import IconBuildingBank from '@tabler/icons-svelte/icons/building-bank';
	import IconCalculator from '@tabler/icons-svelte/icons/calculator';
	import IconChevronRight from '@tabler/icons-svelte/icons/chevron-right';
	import IconClipboardCheck from '@tabler/icons-svelte/icons/clipboard-check';
	import IconFileText from '@tabler/icons-svelte/icons/file-text';
	import IconHome from '@tabler/icons-svelte/icons/home';
	import IconLayoutDashboard from '@tabler/icons-svelte/icons/layout-dashboard';
	import IconReceiptEuro from '@tabler/icons-svelte/icons/receipt-euro';
	import { untrack } from 'svelte';
	import * as Collapsible from '#lib/components/ui/collapsible/index.js';
	import * as Sidebar from '#lib/components/ui/sidebar/index.js';
	import type { UserDTO } from '#lib/modules/auth/types';
	import type { LegalEntityDTO } from '#lib/modules/portfolio/types';
	import EntitySwitcher from './EntitySwitcher.svelte';
	import { isExactRouteActive, isGroupPathActive } from './sidebar-nav';
	import SidebarUserMenu from './SidebarUserMenu.svelte';

	let {
		user = null,
		entities = []
	}: {
		user?: UserDTO | null;
		entities?: LegalEntityDTO[];
	} = $props();

	type TablerIconComponent = typeof IconHome;

	type AppLeafRoute =
		| '/(app)/properties'
		| '/(app)/leases'
		| '/(app)/leases/tenants'
		| '/(app)/leases/templates'
		| '/(app)/rents'
		| '/(app)/rents/cash-journal'
		| '/(app)/inspections'
		| '/(app)/inspections/incidents'
		| '/(app)/inspections/templates'
		| '/(app)/entities'
		| '/(app)/accounting'
		| '/(app)/accounting/journal'
		| '/(app)/accounting/tax';

	type NavSubItem = {
		label: string;
		route: AppLeafRoute;
	};

	type DirectNavItem = {
		kind: 'direct';
		key: string;
		label: string;
		route: AppLeafRoute;
		icon: TablerIconComponent;
	};

	type CollapsibleNavItem = {
		kind: 'collapsible';
		key: 'leases' | 'rents' | 'inspections' | 'accounting';
		label: string;
		baseRoute: '/(app)/leases' | '/(app)/rents' | '/(app)/inspections' | '/(app)/accounting';
		icon: TablerIconComponent;
		items: NavSubItem[];
	};

	type SectionNavItem = DirectNavItem | CollapsibleNavItem;

	const sidebar = Sidebar.useSidebar();

	const menuButtonClass = 'h-9 gap-3 text-sm [&_svg]:size-5 group-data-[collapsible=icon]:size-9!';
	const menuSubButtonClass = 'h-8 text-sm data-[size=md]:text-sm';

	const propertyOperationsItems: SectionNavItem[] = [
		{
			kind: 'direct',
			key: 'properties',
			label: 'Biens & Lots',
			route: '/(app)/properties',
			icon: IconHome
		},
		{
			kind: 'collapsible',
			key: 'leases',
			label: 'Baux & Locataires',
			baseRoute: '/(app)/leases',
			icon: IconFileText,
			items: [
				{ label: 'Baux actifs', route: '/(app)/leases' },
				{ label: 'Locataires', route: '/(app)/leases/tenants' },
				{ label: 'Modèles de documents', route: '/(app)/leases/templates' }
			]
		},
		{
			kind: 'collapsible',
			key: 'rents',
			label: 'Loyers & Relances',
			baseRoute: '/(app)/rents',
			icon: IconReceiptEuro,
			items: [
				{ label: 'Appels & Encaissements', route: '/(app)/rents' },
				{ label: 'Livre de caisse (530)', route: '/(app)/rents/cash-journal' }
			]
		},
		{
			kind: 'collapsible',
			key: 'inspections',
			label: 'États des lieux & Incidents',
			baseRoute: '/(app)/inspections',
			icon: IconClipboardCheck,
			items: [
				{ label: 'États des lieux', route: '/(app)/inspections' },
				{ label: 'Incidents & Sinistres', route: '/(app)/inspections/incidents' },
				{ label: 'Modèles d’EDL', route: '/(app)/inspections/templates' }
			]
		}
	];

	const corporateTaxItems: SectionNavItem[] = [
		{
			kind: 'direct',
			key: 'entities',
			label: 'Entités & Associés',
			route: '/(app)/entities',
			icon: IconBuildingBank
		},
		{
			kind: 'collapsible',
			key: 'accounting',
			label: 'Comptabilité & Fiscalité',
			baseRoute: '/(app)/accounting',
			icon: IconCalculator,
			items: [
				{ label: 'Bilan & Compte de résultat', route: '/(app)/accounting' },
				{ label: 'Journal & Balance PCG', route: '/(app)/accounting/journal' },
				{ label: 'Liasse fiscale & Export FEC', route: '/(app)/accounting/tax' }
			]
		}
	];

	const collapsibleGroups = [...propertyOperationsItems, ...corporateTaxItems].filter(
		(item): item is CollapsibleNavItem => item.kind === 'collapsible'
	);

	let openGroups = $state<Record<CollapsibleNavItem['key'], boolean>>({
		leases: false,
		rents: false,
		inspections: false,
		accounting: false
	});

	$effect(() => {
		const pathname = page.url.pathname;

		untrack(() => {
			for (const group of collapsibleGroups) {
				if (isGroupPathActive(resolve(group.baseRoute), pathname) && !openGroups[group.key]) {
					openGroups[group.key] = true;
				}
			}
		});
	});
</script>

{#snippet renderSectionItems(items: SectionNavItem[])}
	<Sidebar.Menu>
		{#each items as item (item.key)}
			{#if item.kind === 'direct'}
				{@const Icon = item.icon}
				<Sidebar.MenuItem>
					<Sidebar.MenuButton
						class={menuButtonClass}
						tooltipContent={item.label}
						isActive={isGroupPathActive(resolve(item.route), page.url.pathname)}
					>
						{#snippet child({ props })}
							<a href={resolve(item.route)} {...props}>
								<Icon />
								<span>{item.label}</span>
							</a>
						{/snippet}
					</Sidebar.MenuButton>
				</Sidebar.MenuItem>
			{:else}
				{@const Icon = item.icon}
				{@const groupActive = isGroupPathActive(resolve(item.baseRoute), page.url.pathname)}
				<Collapsible.Root bind:open={openGroups[item.key]} class="group/collapsible">
					<Sidebar.MenuItem>
						<Collapsible.Trigger>
							{#snippet child({ props })}
								<Sidebar.MenuButton
									{...props}
									class={menuButtonClass}
									tooltipContent={item.label}
									isActive={groupActive && (!openGroups[item.key] || sidebar.state === 'collapsed')}
									onclick={(event) => {
										if (sidebar.state === 'collapsed' && !sidebar.isMobile) {
											event.preventDefault();
											sidebar.setOpen(true);
											openGroups[item.key] = true;
											return;
										}
										const triggerOnClick = props.onclick as ((e: MouseEvent) => void) | undefined;
										triggerOnClick?.(event);
									}}
								>
									<Icon />
									<span>{item.label}</span>
									<IconChevronRight
										class="ml-auto size-4! transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90"
									/>
								</Sidebar.MenuButton>
							{/snippet}
						</Collapsible.Trigger>

						<Collapsible.Content>
							<Sidebar.MenuSub>
								{#each item.items as subItem (subItem.route)}
									<Sidebar.MenuSubItem>
										<Sidebar.MenuSubButton
											class={menuSubButtonClass}
											isActive={isExactRouteActive(resolve(subItem.route), page.url.pathname)}
										>
											{#snippet child({ props })}
												<a href={resolve(subItem.route)} {...props}>
													<span>{subItem.label}</span>
												</a>
											{/snippet}
										</Sidebar.MenuSubButton>
									</Sidebar.MenuSubItem>
								{/each}
							</Sidebar.MenuSub>
						</Collapsible.Content>
					</Sidebar.MenuItem>
				</Collapsible.Root>
			{/if}
		{/each}
	</Sidebar.Menu>
{/snippet}

<Sidebar.Root collapsible="icon">
	<Sidebar.Header>
		<EntitySwitcher {entities} />
	</Sidebar.Header>

	<Sidebar.Content>
		<Sidebar.Group>
			<Sidebar.GroupContent>
				<Sidebar.Menu>
					<Sidebar.MenuItem>
						<Sidebar.MenuButton
							class={menuButtonClass}
							tooltipContent="Vue d’ensemble"
							isActive={isExactRouteActive(resolve('/(app)'), page.url.pathname)}
						>
							{#snippet child({ props })}
								<a href={resolve('/(app)')} {...props}>
									<IconLayoutDashboard />
									<span>Vue d’ensemble</span>
								</a>
							{/snippet}
						</Sidebar.MenuButton>
					</Sidebar.MenuItem>
				</Sidebar.Menu>
			</Sidebar.GroupContent>
		</Sidebar.Group>

		<Sidebar.Group>
			<Sidebar.GroupLabel class="text-xs font-medium">Biens & Exploitation</Sidebar.GroupLabel>
			<Sidebar.GroupContent>
				{@render renderSectionItems(propertyOperationsItems)}
			</Sidebar.GroupContent>
		</Sidebar.Group>

		<Sidebar.Group>
			<Sidebar.GroupLabel class="text-xs font-medium">Société & Fiscalité</Sidebar.GroupLabel>
			<Sidebar.GroupContent>
				{@render renderSectionItems(corporateTaxItems)}
			</Sidebar.GroupContent>
		</Sidebar.Group>
	</Sidebar.Content>

	<Sidebar.Footer>
		<SidebarUserMenu {user} />
	</Sidebar.Footer>

	<Sidebar.Rail />
</Sidebar.Root>
