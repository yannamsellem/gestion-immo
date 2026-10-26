<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import { cn } from '$lib/utils';

	export let tabs: ReadonlyArray<{ id: string; label: string; badge?: string | number }> = [];
	export let activeTab = '';
	let className = '';
	export { className as class };

	const dispatch = createEventDispatcher<{ change: string }>();

	function selectTab(id: string) {
		activeTab = id;
		dispatch('change', id);
	}
</script>

<div class={cn('space-y-4', className)}>
	<div class="inline-flex h-10 items-center justify-center rounded-lg bg-slate-100 p-1 text-slate-600" role="tablist">
		{#each tabs as tab (tab.id)}
			<button
				type="button"
				role="tab"
				aria-selected={activeTab === tab.id}
				on:click={() => selectTab(tab.id)}
				class={cn(
					'inline-flex items-center justify-center whitespace-nowrap rounded-md px-3 py-1.5 text-sm font-medium transition-all',
					activeTab === tab.id
						? 'bg-white text-slate-950 shadow-sm'
						: 'text-slate-600 hover:text-slate-900'
				)}
			>
				<span>{tab.label}</span>
				{#if tab.badge !== undefined}
					<span class="ml-2 rounded-full bg-slate-200 px-2 py-0.5 text-xs font-semibold text-slate-800">
						{tab.badge}
					</span>
				{/if}
			</button>
		{/each}
	</div>
	<div role="tabpanel">
		<slot {activeTab} />
	</div>
</div>
