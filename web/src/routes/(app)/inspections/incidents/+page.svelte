<script lang="ts">
	import { page } from '$app/state';
	import IconAlertTriangle from '@tabler/icons-svelte/icons/alert-triangle';
	import IconPlus from '@tabler/icons-svelte/icons/plus';
	import PageHeader from '#lib/components/layout/PageHeader.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import type { IncidentDTO } from '#lib/modules/inspection/types';
	import type { PageProps } from './$types';
	import CreateIncidentDialog from './(components)/CreateIncidentDialog.svelte';
	import EditIncidentDialog from './(components)/EditIncidentDialog.svelte';
	import IncidentsTableCard from './(components)/IncidentsTableCard.svelte';

	let { data }: PageProps = $props();

	let createIncidentOpen = $state(false);
	let editIncidentOpen = $state(false);
	let selectedIncidentId = $state<string | null>(null);
	let autoOpenedIncidentId = $state<string | null>(null);
	let autoHandledCreate = $state(false);

	const openIncidentsCount = $derived(
		data.incidents.filter((incident) => incident.status !== 'CLOSED').length
	);
	const selectedIncident = $derived(
		data.incidents.find((incident) => incident.id === selectedIncidentId) ?? null
	);

	$effect(() => {
		if (!autoHandledCreate && page.url.searchParams.get('action') === 'create') {
			createIncidentOpen = true;
			autoHandledCreate = true;
		}
	});

	$effect(() => {
		if (data.incidents.length === 0) return;
		const urlIncidentId = page.url.searchParams.get('incidentId');
		if (!urlIncidentId || autoOpenedIncidentId === urlIncidentId) return;

		const match = data.incidents.find((incident) => incident.id === urlIncidentId);
		if (match) {
			selectedIncidentId = match.id;
			editIncidentOpen = true;
			autoOpenedIncidentId = urlIncidentId;
		}
	});

	function openEditIncident(incident: IncidentDTO): void {
		selectedIncidentId = incident.id;
		editIncidentOpen = true;
	}
</script>

<PageHeader
	title="Incidents, Sinistres & Travaux"
	subtitle="Suivi des déclarations d’assurance, interventions techniques et chiffrage des réparations"
>
	{#snippet actions()}
		{#if openIncidentsCount > 0}
			<Badge variant="destructive">
				{openIncidentsCount} dossier{openIncidentsCount > 1 ? 's' : ''} en cours
			</Badge>
		{/if}
		<Button size="sm" onclick={() => (createIncidentOpen = true)}>
			<IconPlus data-icon="inline-start" />
			Déclarer un incident
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

<IncidentsTableCard
	incidents={data.incidents}
	properties={data.properties}
	leases={data.leases}
	tenants={data.tenants}
	onCreateIncident={() => (createIncidentOpen = true)}
	onEditIncident={openEditIncident}
/>

<CreateIncidentDialog
	bind:open={createIncidentOpen}
	properties={data.properties}
	leases={data.leases}
	tenants={data.tenants}
/>

<EditIncidentDialog bind:open={editIncidentOpen} incident={selectedIncident} />
