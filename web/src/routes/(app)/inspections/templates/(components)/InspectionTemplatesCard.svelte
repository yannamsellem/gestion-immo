<script lang="ts">
	import IconClipboardCheck from '@tabler/icons-svelte/icons/clipboard-check';
	import IconPlus from '@tabler/icons-svelte/icons/plus';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Empty from '#lib/components/ui/empty/index.js';
	import * as Table from '#lib/components/ui/table/index.js';
	import type { InspectionTemplateDTO } from '#lib/modules/inspection/types';

	let {
		templates,
		onCreateTemplate
	}: {
		templates: InspectionTemplateDTO[];
		onCreateTemplate: () => void;
	} = $props();
</script>

{#if templates.length === 0}
	<Card.Root>
		<Card.Content class="py-10">
			<Empty.Root>
				<Empty.Header>
					<Empty.Media variant="icon">
						<IconClipboardCheck class="size-5" />
					</Empty.Media>
					<Empty.Title>Aucun modèle d’état des lieux configuré</Empty.Title>
					<Empty.Description>
						Créez des grilles types par typologie de lot (Studio, T2, T3, Local commercial) pour
						pré-remplir vos états des lieux contradictoires.
					</Empty.Description>
				</Empty.Header>
				<Empty.Content>
					<Button onclick={onCreateTemplate}>
						<IconPlus data-icon="inline-start" />
						Nouveau modèle EDL
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
						<Table.Head>Pièces incluses</Table.Head>
						<Table.Head>Éléments inspectés par défaut</Table.Head>
						<Table.Head class="pr-4 text-right">Dimension</Table.Head>
					</Table.Row>
				</Table.Header>
				<Table.Body>
					{#each templates as template (template.id)}
						<Table.Row>
							<Table.Cell class="pl-4 font-medium">{template.name}</Table.Cell>
							<Table.Cell class="text-xs text-muted-foreground">
								{template.rooms.join(', ')}
							</Table.Cell>
							<Table.Cell class="text-xs text-muted-foreground">
								{template.default_elements?.join(', ') || '—'}
							</Table.Cell>
							<Table.Cell class="pr-4 text-right">
								<Badge variant="outline">{template.rooms.length} pièces</Badge>
							</Table.Cell>
						</Table.Row>
					{/each}
				</Table.Body>
			</Table.Root>
		</Card.Content>
	</Card.Root>
{/if}
