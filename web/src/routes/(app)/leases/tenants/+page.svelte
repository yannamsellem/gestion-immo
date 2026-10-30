<script lang="ts">
	import { page } from '$app/state';
	import IconAlertTriangle from '@tabler/icons-svelte/icons/alert-triangle';
	import IconUserPlus from '@tabler/icons-svelte/icons/user-plus';
	import PageHeader from '#lib/components/layout/PageHeader.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import type { PageProps } from './$types';
	import CreateTenantDialog from './(components)/CreateTenantDialog.svelte';
	import TenantsTableCard from './(components)/TenantsTableCard.svelte';

	let { data }: PageProps = $props();

	let createTenantOpen = $state(false);
	let autoHandledUrl = $state(false);

	$effect(() => {
		if (!autoHandledUrl && page.url.searchParams.get('action') === 'create') {
			createTenantOpen = true;
			autoHandledUrl = true;
		}
	});
</script>

<PageHeader
	title="Locataires & Conformité RGPD"
	subtitle="Annuaire des locataires titulaires, baux rattachés et archivage conforme au RGPD"
>
	{#snippet actions()}
		<Button size="sm" onclick={() => (createTenantOpen = true)}>
			<IconUserPlus data-icon="inline-start" />
			Nouveau locataire
		</Button>
	{/snippet}
</PageHeader>

{#if data.errorMessage}
	<Alert.Root variant="destructive">
		<IconAlertTriangle class="size-4" />
		<Alert.Title>Erreur</Alert.Title>
		<Alert.Description>{data.errorMessage}</Alert.Description>
	</Alert.Root>
{/if}

<TenantsTableCard
	tenants={data.tenants}
	activeLeaseCountByTenantId={data.activeLeaseCountByTenantId}
	onCreateTenant={() => (createTenantOpen = true)}
/>

<CreateTenantDialog bind:open={createTenantOpen} />
