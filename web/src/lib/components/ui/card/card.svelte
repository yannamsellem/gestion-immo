<script lang="ts">
	import { cn } from '$lib/utils';

	export let title = '';
	export let description = '';
	let className = '';
	export { className as class };
</script>

<div
	class={cn('rounded-xl border border-slate-200 bg-white text-slate-950 shadow-sm', className)}
	{...$$restProps}
>
	{#if title || description || $$slots.header}
		<div class="flex flex-col space-y-1.5 border-b border-slate-100 p-6">
			{#if $$slots.header}
				<slot name="header" />
			{:else}
				{#if title}
					<h3 class="text-lg font-semibold leading-none tracking-tight">{title}</h3>
				{/if}
				{#if description}
					<p class="text-sm text-slate-500">{description}</p>
				{/if}
			{/if}
		</div>
	{/if}
	<div class="p-6">
		<slot />
	</div>
	{#if $$slots.footer}
		<div class="flex items-center border-t border-slate-100 p-6 pt-4">
			<slot name="footer" />
		</div>
	{/if}
</div>
