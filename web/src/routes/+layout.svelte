<script lang="ts">
	import '../app.css';
	import { Badge } from '$lib/components/ui/badge';
	import type { LegalEntityDTO, UserDTO } from '$lib/api/client';

	export let data: {
		user?: UserDTO | null;
		entities: readonly LegalEntityDTO[];
		activeEntityId: string;
		negativePositionsCount: number;
	};

	let selectedEntityId = data.activeEntityId;
</script>

<div class="flex min-h-screen bg-slate-50 text-slate-900">
	<!-- Sidebar SSR Navigation -->
	<aside class="flex w-72 flex-col border-r border-slate-200 bg-slate-900 text-slate-100">
		<div class="border-b border-slate-800 p-6">
			<div class="flex items-center justify-between">
				<span class="text-lg font-bold tracking-tight text-white">Gestion Immo</span>
				<Badge variant="secondary" class="bg-blue-500/20 text-blue-200">PCG & MCP</Badge>
			</div>
			<p class="mt-1 text-xs text-slate-400">Patrimoine, Loyers, GLI & Liasse Fiscale</p>

			<div class="mt-4">
				<label for="entity-switcher" class="block text-xs font-medium text-slate-400">
					Structure juridique active
				</label>
				<select
					id="entity-switcher"
					bind:value={selectedEntityId}
					class="mt-1.5 w-full rounded-md border border-slate-700 bg-slate-800 px-3 py-2 text-xs font-medium text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
				>
					{#each data.entities as entity (entity.id)}
						<option value={entity.id}>{entity.name} ({entity.entity_type})</option>
					{/each}
				</select>
			</div>
		</div>

		<nav class="flex-1 space-y-1 p-4 text-sm">
			<a
				href="/"
				class="flex items-center justify-between rounded-lg px-3 py-2.5 font-medium text-slate-200 hover:bg-slate-800 hover:text-white"
			>
				<span>Vue d'ensemble & Caisse</span>
			</a>
			<a
				href="/entities/{selectedEntityId}"
				class="flex items-center justify-between rounded-lg px-3 py-2.5 font-medium text-slate-200 hover:bg-slate-800 hover:text-white"
			>
				<span>Détail Société & Associés (2059-F)</span>
			</a>
			<a
				href="/payments/negative-positions"
				class="flex items-center justify-between rounded-lg px-3 py-2.5 font-medium text-slate-200 hover:bg-slate-800 hover:text-white"
			>
				<span>Impayés & Dossiers GLI</span>
				{#if data.negativePositionsCount > 0}
					<Badge variant="destructive">{data.negativePositionsCount}</Badge>
				{/if}
			</a>
			<a
				href="/inspections/insp_demo_01"
				class="flex items-center justify-between rounded-lg px-3 py-2.5 font-medium text-slate-200 hover:bg-slate-800 hover:text-white"
			>
				<span>États des Lieux & MCP</span>
			</a>
			<a
				href="/accounting/{selectedEntityId}/statements"
				class="flex items-center justify-between rounded-lg px-3 py-2.5 font-medium text-slate-200 hover:bg-slate-800 hover:text-white"
			>
				<span>Comptes Annuels & Liasse</span>
				<Badge variant="outline" class="border-emerald-500/40 bg-emerald-500/10 text-emerald-300">
					Interactif
				</Badge>
			</a>

			{#if data.user?.role === 'SUPER_ADMIN'}
				<div class="pt-4">
					<p class="px-3 text-xs font-semibold uppercase tracking-wider text-slate-500">
						Super Administration
					</p>
					<a
						href="/admin/users"
						class="mt-1 flex items-center justify-between rounded-lg px-3 py-2.5 font-medium text-amber-300 hover:bg-slate-800"
					>
						<span>Validation des inscriptions</span>
					</a>
				</div>
			{/if}
		</nav>

		<div class="border-t border-slate-800 p-4 text-xs text-slate-400">
			<div class="flex items-center justify-between">
				<div>
					<p class="font-medium text-slate-200">{data.user?.full_name ?? 'Invité'}</p>
					<p>{data.user?.email ?? ''}</p>
				</div>
				<a href="/login" class="text-blue-400 hover:underline">Connexion</a>
			</div>
		</div>
	</aside>

	<!-- Main Content Area -->
	<main class="flex-1 overflow-y-auto p-8">
		<slot />
	</main>
</div>
