<script lang="ts">
	import { resetMode, setMode } from 'mode-watcher';
	import IconMoon from '@tabler/icons-svelte/icons/moon';
	import IconSun from '@tabler/icons-svelte/icons/sun';
	import { buttonVariants } from '#lib/components/ui/button/index.js';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.js';
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger
		class={buttonVariants({ variant: 'outline', size: 'icon' })}
		aria-label="Basculer le thème clair ou sombre"
	>
		<IconSun class="scale-100 rotate-0 transition-all dark:scale-0 dark:-rotate-90" />
		<IconMoon class="absolute scale-0 rotate-90 transition-all dark:scale-100 dark:rotate-0" />
		<span class="sr-only">Changer le thème</span>
	</DropdownMenu.Trigger>
	<DropdownMenu.Content align="end">
		<DropdownMenu.Group>
			<DropdownMenu.Item onclick={() => setMode('light')}>Mode clair</DropdownMenu.Item>
			<DropdownMenu.Item onclick={() => setMode('dark')}>Mode sombre</DropdownMenu.Item>
			<DropdownMenu.Item onclick={() => resetMode()}>Système</DropdownMenu.Item>
		</DropdownMenu.Group>
	</DropdownMenu.Content>
</DropdownMenu.Root>
