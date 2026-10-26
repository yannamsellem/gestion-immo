<script lang="ts">
	import { cn } from '$lib/utils';

	export let variant: 'default' | 'destructive' | 'outline' | 'secondary' | 'ghost' | 'link' = 'default';
	export let size: 'default' | 'sm' | 'lg' | 'icon' = 'default';
	export let type: 'button' | 'submit' | 'reset' = 'button';
	export let disabled = false;
	let className = '';
	export { className as class };

	const variantStyles: Record<typeof variant, string> = {
		default: 'bg-blue-600 text-white hover:bg-blue-700 shadow-sm',
		destructive: 'bg-red-600 text-white hover:bg-red-700 shadow-sm',
		outline: 'border border-slate-300 bg-white text-slate-900 hover:bg-slate-50',
		secondary: 'bg-slate-100 text-slate-900 hover:bg-slate-200',
		ghost: 'text-slate-700 hover:bg-slate-100',
		link: 'text-blue-600 underline-offset-4 hover:underline'
	};

	const sizeStyles: Record<typeof size, string> = {
		default: 'h-10 px-4 py-2 text-sm',
		sm: 'h-8 px-3 text-xs',
		lg: 'h-11 px-6 text-base',
		icon: 'h-10 w-10'
	};
</script>

<button
	{type}
	{disabled}
	on:click
	class={cn(
		'inline-flex items-center justify-center rounded-md font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 disabled:pointer-events-none disabled:opacity-50',
		variantStyles[variant],
		sizeStyles[size],
		className
	)}
	{...$$restProps}
>
	<slot />
</button>
