<script lang="ts">
	import './layout.css';
	import { onMount } from 'svelte';
	import { ModeWatcher } from 'mode-watcher';
	import favicon from '#lib/assets/favicon.svg';
	import { Toaster } from '#lib/components/ui/sonner/index.js';
	import { workspaceStore } from '#lib/core/workspace/workspace.svelte';
	import { authStore } from '#lib/modules/auth/auth-store.svelte';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();

	onMount(() => {
		authStore.initialize();
		workspaceStore.initialize();
	});
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<title>Gestion Immo — Pilotage Patrimonial & Locatif</title>
</svelte:head>

<ModeWatcher />
<Toaster richColors closeButton />

{@render children()}
