<script lang="ts">
	import { page } from '$app/state';
	import IconAlertTriangle from '@tabler/icons-svelte/icons/alert-triangle';
	import IconClipboardCheck from '@tabler/icons-svelte/icons/clipboard-check';
	import PageHeader from '#lib/components/layout/PageHeader.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import type { PageProps } from './$types';
	import CreateInspectionTemplateDialog from './(components)/CreateInspectionTemplateDialog.svelte';
	import InspectionTemplatesCard from './(components)/InspectionTemplatesCard.svelte';

	let { data }: PageProps = $props();

	let createTemplateOpen = $state(false);
	let autoHandledUrl = $state(false);

	$effect(() => {
		if (!autoHandledUrl && page.url.searchParams.get('action') === 'create') {
			createTemplateOpen = true;
			autoHandledUrl = true;
		}
	});
</script>

<PageHeader
	title="Modèles d’états des lieux"
	subtitle="Grilles types réutilisables par typologie de lot (Studio, T2, T3, Maison, Local commercial)"
>
	{#snippet actions()}
		<Button size="sm" onclick={() => (createTemplateOpen = true)}>
			<IconClipboardCheck data-icon="inline-start" />
			Nouveau modèle EDL
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

<InspectionTemplatesCard
	templates={data.templates}
	onCreateTemplate={() => (createTemplateOpen = true)}
/>

<CreateInspectionTemplateDialog bind:open={createTemplateOpen} />
