<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import { cn } from '$lib/utils';

	export let open = false;
	export let title = '';
	export let description = '';
	let className = '';
	export { className as class };

	const dispatch = createEventDispatcher<{ close: void }>();

	function handleClose() {
		open = false;
		dispatch('close');
	}
</script>

{#if open}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/60 p-4 backdrop-blur-xs"
		role="dialog"
		aria-modal="true"
		aria-label={title}
	>
		<div
			class={cn(
				'relative w-full max-w-lg rounded-xl border border-slate-200 bg-white p-6 shadow-xl',
				className
			)}
		>
			<div class="flex items-start justify-between border-b border-slate-100 pb-4">
				<div>
					{#if title}
						<h2 class="text-lg font-semibold text-slate-900">{title}</h2>
					{/if}
					{#if description}
						<p class="mt-1 text-sm text-slate-500">{description}</p>
					{/if}
				</div>
				<button
					type="button"
					on:click={handleClose}
					class="rounded-md p-1 text-slate-400 hover:bg-slate-100 hover:text-slate-700"
					aria-label="Fermer"
				>
					✕
				</button>
			</div>
			<div class="mt-4">
				<slot />
			</div>
			{#if $$slots.footer}
				<div class="mt-6 flex justify-end gap-2 border-t border-slate-100 pt-4">
					<slot name="footer" />
				</div>
			{/if}
		</div>
	</div>
{/if}
