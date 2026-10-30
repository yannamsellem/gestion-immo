<script lang="ts">
	import PageHeader from '#lib/components/layout/PageHeader.svelte';
	import McpTokensCard from './(components)/McpTokensCard.svelte';
	import PendingUsersGovernanceCard from './(components)/PendingUsersGovernanceCard.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
</script>

<PageHeader
	title="Administration, Sécurité & Accès MCP"
	subtitle="Génération de jetons d’accès MCP longue durée (Claude Desktop / Agents IA) et gouvernance des gestionnaires"
/>

<div class="grid gap-6 lg:grid-cols-12">
	<McpTokensCard />
	<PendingUsersGovernanceCard
		isSuperAdmin={data.isSuperAdmin}
		userRole={data.user?.role ?? 'MANAGER'}
		pendingUsers={data.pendingUsers}
		pendingError={data.pendingError}
	/>
</div>
