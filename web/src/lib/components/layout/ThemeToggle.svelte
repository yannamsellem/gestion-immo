<script lang="ts">
	import IconDeviceDesktop from '@tabler/icons-svelte/icons/device-desktop';
	import IconMoon from '@tabler/icons-svelte/icons/moon';
	import IconSun from '@tabler/icons-svelte/icons/sun';
	import { resetMode, setMode, systemPrefersMode, userPrefersMode, mode } from 'mode-watcher';
	import { buttonVariants } from '#lib/components/ui/button/index.js';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.js';
	import { applyThemePreference, formatSystemThemeLabel } from './theme-toggle';
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger
		class={buttonVariants({ variant: 'outline', size: 'icon' })}
		aria-label="Choisir le thème d’affichage"
	>
		{#if mode.current === 'light'}
			<IconSun />
		{:else}
			<IconMoon />
		{/if}
		<span class="sr-only">Choisir le thème d’affichage</span>
	</DropdownMenu.Trigger>
	<DropdownMenu.Content align="end" class="min-w-44">
		<DropdownMenu.Group>
			<DropdownMenu.Label>Thème d’affichage</DropdownMenu.Label>
		</DropdownMenu.Group>
		<DropdownMenu.Separator />
		<DropdownMenu.RadioGroup
			value={userPrefersMode.current}
			onValueChange={(next) => {
				applyThemePreference(next, { setMode, resetMode });
			}}
		>
			<DropdownMenu.RadioItem value="light" closeOnSelect>
				<IconSun />
				<span>Mode clair</span>
			</DropdownMenu.RadioItem>
			<DropdownMenu.RadioItem value="dark" closeOnSelect>
				<IconMoon />
				<span>Mode sombre</span>
			</DropdownMenu.RadioItem>
			<DropdownMenu.RadioItem value="system" closeOnSelect>
				<IconDeviceDesktop />
				<span>{formatSystemThemeLabel(systemPrefersMode.current)}</span>
			</DropdownMenu.RadioItem>
		</DropdownMenu.RadioGroup>
	</DropdownMenu.Content>
</DropdownMenu.Root>
