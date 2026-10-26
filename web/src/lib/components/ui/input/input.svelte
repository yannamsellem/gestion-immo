<script lang="ts">
	import { cn } from '$lib/utils';

	export let value: string | number = '';
	export let type = 'text';
	export let placeholder = '';
	export let disabled = false;
	export let required = false;
	export let id = '';
	export let name = '';
	let className = '';
	export { className as class };
</script>

<input
	{id}
	{name}
	{type}
	{placeholder}
	{disabled}
	{required}
	bind:value
	on:input
	on:change
	class={cn(
		'flex h-10 w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 placeholder:text-slate-400 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 disabled:cursor-not-allowed disabled:opacity-50',
		className
	)}
	{...$$restProps}
/>
