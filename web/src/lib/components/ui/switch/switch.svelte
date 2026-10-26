<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import { cn } from '$lib/utils';

	export let checked = true;
	export let disabled = false;
	export let ariaLabel = 'Activer ou désactiver la ligne';
	let className = '';
	export { className as class };

	const dispatch = createEventDispatcher<{ change: boolean }>();

	function handleToggle() {
		if (disabled) return;
		checked = !checked;
		dispatch('change', checked);
	}
</script>

<button
	type="button"
	role="switch"
	aria-checked={checked}
	aria-label={ariaLabel}
	{disabled}
	on:click={handleToggle}
	class={cn(
		'peer inline-flex h-6 w-11 shrink-0 cursor-pointer items-center rounded-full border-2 border-transparent transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 disabled:cursor-not-allowed disabled:opacity-50',
		checked ? 'bg-blue-600' : 'bg-slate-200',
		className
	)}
	{...$$restProps}
>
	<span
		class={cn(
			'pointer-events-none block h-5 w-5 rounded-full bg-white shadow-lg ring-0 transition-transform',
			checked ? 'translate-x-5' : 'translate-x-0'
		)}
	></span>
</button>
