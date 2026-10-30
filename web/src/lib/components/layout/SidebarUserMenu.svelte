<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import IconLogout from '@tabler/icons-svelte/icons/logout';
	import IconSelector from '@tabler/icons-svelte/icons/selector';
	import IconShieldLock from '@tabler/icons-svelte/icons/shield-lock';
	import * as Avatar from '#lib/components/ui/avatar/index.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.js';
	import * as Sidebar from '#lib/components/ui/sidebar/index.js';
	import type { UserDTO } from '#lib/modules/auth/types';
	import { formatUserInitials } from './sidebar-nav';

	let {
		user
	}: {
		user: UserDTO | null;
	} = $props();

	const sidebar = Sidebar.useSidebar();

	let logoutFormEl = $state<HTMLFormElement | null>(null);

	const displayName = $derived(user?.full_name || 'Utilisateur');
	const displayEmail = $derived(user?.email || '');
	const initials = $derived(formatUserInitials(user?.full_name));
</script>

<form
	bind:this={logoutFormEl}
	method="POST"
	action={resolve('/(auth)/logout')}
	use:enhance
	class="hidden"
></form>

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
						<Avatar.Root class="size-8 rounded-lg">
							<Avatar.Fallback class="rounded-lg text-xs font-semibold">
								{initials}
							</Avatar.Fallback>
						</Avatar.Root>
						<div class="grid flex-1 text-left text-sm leading-tight">
							<span class="truncate font-semibold">{displayName}</span>
							<span class="truncate text-xs text-muted-foreground">{displayEmail}</span>
						</div>
						<IconSelector class="ml-auto size-4" />
					</Sidebar.MenuButton>
				{/snippet}
			</DropdownMenu.Trigger>

			<DropdownMenu.Content
				class="w-(--bits-dropdown-menu-anchor-width) min-w-56 rounded-lg"
				side={sidebar.isMobile ? 'bottom' : 'right'}
				align="end"
				sideOffset={4}
			>
				<DropdownMenu.Group>
					<DropdownMenu.Label class="p-0 font-normal">
						<div class="flex items-center gap-2.5 px-2 py-1.5 text-left text-sm">
							<Avatar.Root class="size-8 rounded-lg">
								<Avatar.Fallback class="rounded-lg text-xs font-semibold">
									{initials}
								</Avatar.Fallback>
							</Avatar.Root>
							<div class="grid flex-1 text-left text-sm leading-tight">
								<div class="flex items-center gap-1.5">
									<span class="truncate font-semibold">{displayName}</span>
									{#if user?.role}
										<Badge variant="secondary" class="px-1.5 py-0 text-[10px]">
											{user.role}
										</Badge>
									{/if}
								</div>
								<span class="truncate text-xs text-muted-foreground">{displayEmail}</span>
							</div>
						</div>
					</DropdownMenu.Label>
				</DropdownMenu.Group>

				<DropdownMenu.Separator />

				<DropdownMenu.Group>
					<DropdownMenu.Item class="cursor-pointer gap-2.5 p-2 text-sm">
						{#snippet child({ props })}
							<a href={resolve('/(app)/admin')} {...props}>
								<IconShieldLock class="size-4" />
								<span>Administration & Accès MCP</span>
							</a>
						{/snippet}
					</DropdownMenu.Item>
				</DropdownMenu.Group>

				<DropdownMenu.Separator />

				<DropdownMenu.Item
					variant="destructive"
					class="cursor-pointer gap-2.5 p-2 text-sm"
					onclick={() => {
						logoutFormEl?.requestSubmit();
					}}
				>
					<IconLogout class="size-4" />
					<span>Se déconnecter</span>
				</DropdownMenu.Item>
			</DropdownMenu.Content>
		</DropdownMenu.Root>
	</Sidebar.MenuItem>
</Sidebar.Menu>
