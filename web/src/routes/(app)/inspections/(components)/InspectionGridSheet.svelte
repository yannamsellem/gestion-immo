<script lang="ts">
	import { enhance } from '$app/forms';
	import { invalidateAll } from '$app/navigation';
	import IconDownload from '@tabler/icons-svelte/icons/download';
	import IconFileText from '@tabler/icons-svelte/icons/file-text';
	import { toast } from 'svelte-sonner';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import * as Select from '#lib/components/ui/select/index.js';
	import * as Sheet from '#lib/components/ui/sheet/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import {
		CONDITION_STATE_OPTIONS,
		formatConditionStateLabel,
		formatInspectionTypeLabel
	} from '#lib/modules/inspection/labels';
	import type {
		InspectionDTO,
		InspectionItemInput,
		InspectionReportDTO,
		InspectionTemplateDTO
	} from '#lib/modules/inspection/types';

	let {
		open = $bindable(false),
		inspection,
		templates
	}: {
		open?: boolean;
		inspection: InspectionDTO | null;
		templates: InspectionTemplateDTO[];
	} = $props();

	let items = $state<InspectionItemInput[]>([]);
	let generatedReport = $state<InspectionReportDTO | null>(null);
	let isSavingItems = $state(false);
	let isGeneratingPdf = $state(false);
	let syncedInspectionId = $state<string | null>(null);

	function buildInitialItems(target: InspectionDTO): InspectionItemInput[] {
		if (target.items && target.items.length > 0) {
			return target.items.map((item) => ({
				room_name: item.room_name,
				element_name: item.element_name,
				condition: item.condition,
				observation: item.observation ?? ''
			}));
		}
		const template = templates.find((tpl) => tpl.id === target.template_id);
		const rooms = template?.rooms ?? ['Séjour', 'Cuisine', 'Chambre', 'Salle de bain'];
		const elements = template?.default_elements?.length
			? template.default_elements.slice(0, 3)
			: ['Sol', 'Murs', 'Équipements'];
		const defaults: InspectionItemInput[] = [];
		for (const room of rooms) {
			for (const element of elements) {
				defaults.push({
					room_name: room,
					element_name: element,
					condition: 'GOOD',
					observation: ''
				});
			}
		}
		return defaults;
	}

	$effect(() => {
		if (open && inspection && syncedInspectionId !== inspection.id) {
			items = buildInitialItems(inspection);
			generatedReport = null;
			syncedInspectionId = inspection.id;
		}
		if (!open) {
			syncedInspectionId = null;
		}
	});
</script>

<Sheet.Root bind:open>
	<Sheet.Content side="right" class="w-full overflow-y-auto sm:max-w-xl">
		<Sheet.Header class="border-b pb-4">
			<Sheet.Title>
				Grille d’état des lieux {inspection ? formatInspectionTypeLabel(inspection.type) : ''}
			</Sheet.Title>
			<Sheet.Description>
				Qualifiez chaque équipement pièce par pièce puis générez le rapport PDF contradictoire.
			</Sheet.Description>
		</Sheet.Header>

		{#if inspection}
			<div class="flex flex-col gap-4 p-4">
				<div class="flex flex-col gap-2.5">
					{#each items as item, idx (`${item.room_name}-${item.element_name}-${idx}`)}
						<div class="grid gap-2 rounded-md border p-2.5 sm:grid-cols-12 sm:items-center">
							<div class="sm:col-span-4">
								<p class="text-xs font-semibold">{item.room_name}</p>
								<p class="text-[11px] text-muted-foreground">{item.element_name}</p>
							</div>
							<div class="sm:col-span-4">
								<Select.Root
									type="single"
									value={item.condition}
									onValueChange={(next) => {
										if (next) items[idx].condition = next;
									}}
								>
									<Select.Trigger class="w-full">
										{formatConditionStateLabel(item.condition)}
									</Select.Trigger>
									<Select.Content>
										<Select.Group>
											{#each CONDITION_STATE_OPTIONS as option (option.value)}
												<Select.Item value={option.value} label={option.label}>
													{option.label}
												</Select.Item>
											{/each}
										</Select.Group>
									</Select.Content>
								</Select.Root>
							</div>
							<div class="sm:col-span-4">
								<Input
									placeholder="Observation..."
									value={item.observation ?? ''}
									oninput={(event) => {
										items[idx].observation = (event.currentTarget as HTMLInputElement).value;
									}}
								/>
							</div>
						</div>
					{/each}
				</div>

				<div class="flex flex-wrap items-center gap-2">
					<form
						method="POST"
						action="?/saveInspectionItems"
						use:enhance={() => {
							isSavingItems = true;
							return async ({ result }) => {
								isSavingItems = false;
								if (result.type === 'success') {
									const payload = result.data as { successMessage?: string } | undefined;
									toast.success(payload?.successMessage ?? 'Relevé pièce par pièce enregistré.');
									await invalidateAll();
									return;
								}
								if (result.type === 'failure') {
									const payload = result.data as { actionError?: string } | undefined;
									toast.error(payload?.actionError ?? 'Impossible d’enregistrer les éléments.');
									return;
								}
								if (result.type === 'error') {
									toast.error(result.error?.message ?? 'Erreur serveur lors de l’enregistrement.');
								}
							};
						}}
					>
						<input type="hidden" name="inspection_id" value={inspection.id} />
						<input type="hidden" name="items_json" value={JSON.stringify(items)} />
						<Button type="submit" variant="secondary" disabled={isSavingItems}>
							{#if isSavingItems}
								<Spinner data-icon="inline-start" />
							{/if}
							Sauvegarder la grille
						</Button>
					</form>

					<form
						method="POST"
						action="?/generatePdf"
						use:enhance={() => {
							isGeneratingPdf = true;
							return async ({ result }) => {
								isGeneratingPdf = false;
								if (result.type === 'success') {
									const payload = result.data as
										{ generatedReport?: InspectionReportDTO; successMessage?: string } | undefined;
									if (payload?.generatedReport) {
										generatedReport = payload.generatedReport;
									}
									toast.success(payload?.successMessage ?? 'Rapport PDF contradictoire généré.');
									await invalidateAll();
									return;
								}
								if (result.type === 'failure') {
									const payload = result.data as { actionError?: string } | undefined;
									toast.error(payload?.actionError ?? 'Échec de la génération du rapport PDF.');
									return;
								}
								if (result.type === 'error') {
									toast.error(result.error?.message ?? 'Erreur serveur lors de la génération PDF.');
								}
							};
						}}
					>
						<input type="hidden" name="inspection_id" value={inspection.id} />
						<input type="hidden" name="items_json" value={JSON.stringify(items)} />
						<Button type="submit" disabled={isGeneratingPdf}>
							{#if isGeneratingPdf}
								<Spinner data-icon="inline-start" />
							{:else}
								<IconFileText data-icon="inline-start" />
							{/if}
							Générer le PDF contradictoire
						</Button>
					</form>
				</div>

				{#if generatedReport}
					<div class="flex flex-col gap-2 rounded-md border bg-muted/40 p-3 text-xs">
						<div class="flex items-center justify-between">
							<span class="font-semibold">Rapport PDF généré</span>
							{#if generatedReport.download_url}
								<Button
									size="sm"
									variant="outline"
									href={generatedReport.download_url}
									target="_blank"
									rel="noreferrer"
								>
									<IconDownload data-icon="inline-start" />
									Télécharger le PDF
								</Button>
							{/if}
						</div>

						{#if generatedReport.degradations && generatedReport.degradations.length > 0}
							<p class="font-medium text-destructive">
								{generatedReport.degradations.length} dégradation(s) retenue(s) entre l’entrée et la sortie.
							</p>
						{/if}
					</div>
				{:else if inspection.report_download_url}
					<div class="flex items-center justify-between rounded-md border bg-muted/40 p-3 text-xs">
						<span class="font-semibold">Rapport PDF archivé disponible</span>
						<Button
							size="sm"
							variant="outline"
							href={inspection.report_download_url}
							target="_blank"
							rel="noreferrer"
						>
							<IconDownload data-icon="inline-start" />
							Télécharger le PDF
						</Button>
					</div>
				{/if}
			</div>
		{/if}
	</Sheet.Content>
</Sheet.Root>
