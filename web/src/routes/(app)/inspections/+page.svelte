<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import IconAlertTriangle from '@tabler/icons-svelte/icons/alert-triangle';
	import IconClipboardCheck from '@tabler/icons-svelte/icons/clipboard-check';
	import IconPlus from '@tabler/icons-svelte/icons/plus';
	import IconTool from '@tabler/icons-svelte/icons/tool';
	import PageHeader from '#lib/components/layout/PageHeader.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import type { InspectionDTO } from '#lib/modules/inspection/types';
	import type { PageProps } from './$types';
	import CreateInspectionDialog from './(components)/CreateInspectionDialog.svelte';
	import InspectionGridSheet from './(components)/InspectionGridSheet.svelte';
	import InspectionsListCard from './(components)/InspectionsListCard.svelte';

	let { data }: PageProps = $props();

	let createInspectionOpen = $state(false);
	let gridSheetOpen = $state(false);
	let selectedInspectionId = $state<string | null>(null);
	let transientCreatedInspection = $state<InspectionDTO | null>(null);
	let autoHandledUrl = $state(false);

	const activeInspection = $derived(
		data.inspections.find((insp) => insp.id === selectedInspectionId) ??
			(transientCreatedInspection?.id === selectedInspectionId ? transientCreatedInspection : null)
	);

	$effect(() => {
		if (!autoHandledUrl && page.url.searchParams.get('action') === 'create') {
			createInspectionOpen = true;
			autoHandledUrl = true;
		}
	});

	function openInspectionGrid(inspection: InspectionDTO): void {
		transientCreatedInspection = inspection;
		selectedInspectionId = inspection.id;
		gridSheetOpen = true;
	}
</script>

<PageHeader
	title="États des lieux contradictoires"
	subtitle="Relevés d’entrée et de sortie pièce par pièce avec comparatif automatique des dégradations et génération PDF"
>
	{#snippet actions()}
		<Button size="sm" variant="outline" href={resolve('/(app)/inspections/templates')}>
			<IconClipboardCheck data-icon="inline-start" />
			Modèles d’EDL ({data.templates.length})
		</Button>
		<Button size="sm" variant="outline" href={resolve('/(app)/inspections/incidents')}>
			<IconTool data-icon="inline-start" />
			Incidents & Sinistres
		</Button>
		<Button size="sm" onclick={() => (createInspectionOpen = true)}>
			<IconPlus data-icon="inline-start" />
			Nouvel état des lieux
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

<InspectionsListCard
	inspections={data.inspections}
	leases={data.leases}
	tenants={data.tenants}
	templates={data.templates}
	onCreateInspection={() => (createInspectionOpen = true)}
	onOpenInspection={openInspectionGrid}
/>

<CreateInspectionDialog
	bind:open={createInspectionOpen}
	leases={data.leases}
	tenants={data.tenants}
	templates={data.templates}
	inspections={data.inspections}
	onCreated={openInspectionGrid}
/>

<InspectionGridSheet
	bind:open={gridSheetOpen}
	inspection={activeInspection}
	templates={data.templates}
/>
