import { invalidateAll } from '$app/navigation';
import { getCurrentPeriodMonth } from '#lib/core/formatters/date';
import type { LegalEntityDTO } from '#lib/modules/portfolio/types';

const WORKSPACE_ENTITY_COOKIE = 'gi_entity_id';
const WORKSPACE_PERIOD_COOKIE = 'gi_period_month';

class WorkspaceStore {
	selectedEntityId = $state<'ALL' | string>('ALL');
	selectedPeriodMonth = $state<string>(getCurrentPeriodMonth());
	entities = $state<LegalEntityDTO[]>([]);
	loadingEntities = $state<boolean>(false);

	readonly activeEntity = $derived(
		this.selectedEntityId === 'ALL'
			? null
			: (this.entities.find((entity) => entity.id === this.selectedEntityId) ?? null)
	);

	hydrateFromServer(params: {
		entities: LegalEntityDTO[];
		selectedEntityId: 'ALL' | string;
		selectedPeriodMonth: string;
	}): void {
		this.entities = params.entities;
		this.selectedPeriodMonth = params.selectedPeriodMonth;
		if (
			params.selectedEntityId !== 'ALL' &&
			params.entities.length > 0 &&
			!params.entities.some((entity) => entity.id === params.selectedEntityId)
		) {
			this.selectedEntityId = 'ALL';
		} else {
			this.selectedEntityId = params.selectedEntityId;
		}
	}

	async setEntityId(entityId: 'ALL' | string): Promise<void> {
		this.selectedEntityId = entityId;
		if (typeof document !== 'undefined') {
			document.cookie = `${WORKSPACE_ENTITY_COOKIE}=${encodeURIComponent(entityId)}; path=/; max-age=31536000; SameSite=Lax`;
			await invalidateAll();
		}
	}

	async setPeriodMonth(periodMonth: string): Promise<void> {
		if (/^\d{4}-\d{2}$/.test(periodMonth)) {
			this.selectedPeriodMonth = periodMonth;
			if (typeof document !== 'undefined') {
				document.cookie = `${WORKSPACE_PERIOD_COOKIE}=${encodeURIComponent(periodMonth)}; path=/; max-age=31536000; SameSite=Lax`;
				await invalidateAll();
			}
		}
	}

	setEntities(entities: LegalEntityDTO[]): void {
		this.entities = entities;
	}
}

export const workspaceStore = new WorkspaceStore();
