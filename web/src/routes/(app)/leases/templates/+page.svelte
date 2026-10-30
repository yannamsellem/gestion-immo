<script lang="ts">
	import { page } from '$app/state';
	import IconAlertTriangle from '@tabler/icons-svelte/icons/alert-triangle';
	import IconFileDescription from '@tabler/icons-svelte/icons/file-description';
	import PageHeader from '#lib/components/layout/PageHeader.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import type { PageProps } from './$types';
	import DocumentTemplatesCard from './(components)/DocumentTemplatesCard.svelte';
	import RegisterTemplateDialog from './(components)/RegisterTemplateDialog.svelte';

	let { data }: PageProps = $props();

	let registerTemplateOpen = $state(false);
	let autoHandledUrl = $state(false);

	$effect(() => {
		if (!autoHandledUrl && page.url.searchParams.get('action') === 'create') {
			registerTemplateOpen = true;
			autoHandledUrl = true;
		}
	});
</script>

<PageHeader
	title="Modèles de documents (.docx / PDF)"
	subtitle="Gabarits Word .docx et HTML convertis en PDF pour contrats de bail, quittances, relances et déclarations GLI"
>
	{#snippet actions()}
		<Button size="sm" onclick={() => (registerTemplateOpen = true)}>
			<IconFileDescription data-icon="inline-start" />
			Nouveau modèle (.docx / HTML)
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

<DocumentTemplatesCard
	templates={data.templates}
	onRegisterTemplate={() => (registerTemplateOpen = true)}
/>

<RegisterTemplateDialog bind:open={registerTemplateOpen} />
