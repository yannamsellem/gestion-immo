<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		title,
		subtitle,
		actions
	}: {
		title: string;
		subtitle?: string;
		actions?: Snippet;
	} = $props();
</script>

<div class="flex flex-col justify-between gap-3 border-b pb-4 sm:flex-row sm:items-center">
	<div class="flex flex-col gap-1">
		<h1 class="text-lg font-semibold tracking-tight">{title}</h1>
		{#if subtitle}
			<p class="text-xs text-muted-foreground">{subtitle}</p>
		{/if}
	</div>
	{#if actions}
		<div class="flex flex-wrap items-center gap-2">
			{@render actions()}
		</div>
	{/if}
</div>
