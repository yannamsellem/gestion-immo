<script lang="ts">
	import IconFileDescription from '@tabler/icons-svelte/icons/file-description';
	import IconPlus from '@tabler/icons-svelte/icons/plus';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Empty from '#lib/components/ui/empty/index.js';
	import * as Table from '#lib/components/ui/table/index.js';
	import { formatIsoDateFr } from '#lib/core/formatters/date';
	import { formatTemplateTypeLabel } from '#lib/modules/document/labels';
	import type { DocumentTemplateDTO } from '#lib/modules/document/types';

	let {
		templates,
		onRegisterTemplate
	}: {
		templates: DocumentTemplateDTO[];
		onRegisterTemplate: () => void;
	} = $props();
</script>

{#if templates.length === 0}
	<Card.Root>
		<Card.Content class="py-10">
			<Empty.Root>
				<Empty.Header>
					<Empty.Media variant="icon">
						<IconFileDescription class="size-5" />
					</Empty.Media>
					<Empty.Title>Aucun modèle documentaire enregistré</Empty.Title>
					<Empty.Description>
						Ajoutez un modèle Word <code>.docx</code> ou un gabarit HTML pour générer baux, quittances,
						mises en demeure et dossiers sinistre GLI.
					</Empty.Description>
				</Empty.Header>
				<Empty.Content>
					<Button onclick={onRegisterTemplate}>
						<IconPlus data-icon="inline-start" />
						Nouveau modèle (.docx / HTML)
					</Button>
				</Empty.Content>
			</Empty.Root>
		</Card.Content>
	</Card.Root>
{:else}
	<Card.Root>
		<Card.Content class="p-0">
			<Table.Root>
				<Table.Header>
					<Table.Row>
						<Table.Head class="pl-4">Nom du modèle</Table.Head>
						<Table.Head>Catégorie</Table.Head>
						<Table.Head>Format moteur</Table.Head>
						<Table.Head class="pr-4 text-right">Créé le</Table.Head>
					</Table.Row>
				</Table.Header>
				<Table.Body>
					{#each templates as template (template.id)}
						<Table.Row>
							<Table.Cell class="pl-4 font-medium">{template.name}</Table.Cell>
							<Table.Cell>
								<Badge variant="outline">
									{formatTemplateTypeLabel(template.template_type)}
								</Badge>
							</Table.Cell>
							<Table.Cell>
								<Badge variant="secondary">{template.format}</Badge>
							</Table.Cell>
							<Table.Cell class="pr-4 text-right text-xs">
								{formatIsoDateFr(template.created_at)}
							</Table.Cell>
						</Table.Row>
					{/each}
				</Table.Body>
			</Table.Root>
		</Card.Content>
	</Card.Root>
{/if}
