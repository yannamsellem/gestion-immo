<script lang="ts">
	import { cn } from '$lib/utils';

	export let variant: 'default' | 'secondary' | 'destructive' | 'outline' | 'success' | 'warning' = 'default';
	let className = '';
	export { className as class };

	const variantStyles: Record<typeof variant, string> = {
		default: 'bg-blue-600 text-white border-transparent',
		secondary: 'bg-slate-100 text-slate-900 border-transparent',
		destructive: 'bg-red-100 text-red-800 border-red-200',
		outline: 'border-slate-300 text-slate-700 bg-white',
		success: 'bg-emerald-100 text-emerald-800 border-emerald-200',
		warning: 'bg-amber-100 text-amber-800 border-amber-200'
	};
</script>

<span
	class={cn(
		'inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold transition-colors',
		variantStyles[variant],
		className
	)}
	{...$$restProps}
>
	<slot />
</span>
