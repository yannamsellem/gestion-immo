import { getCurrentPeriodMonth } from '#lib/core/formatters/date';
import type { LegalEntityDTO } from '#lib/modules/portfolio/types';

const WORKSPACE_ENTITY_KEY = 'gestion_immo_selected_entity_id';

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

	initialize(): void {
		if (typeof window === 'undefined') {
			return;
		}
		const saved = window.localStorage.getItem(WORKSPACE_ENTITY_KEY);
		if (saved && saved.trim() !== '') {
			this.selectedEntityId = saved;
		}
	}

	setEntityId(entityId: 'ALL' | string): void {
		this.selectedEntityId = entityId;
		if (typeof window !== 'undefined') {
			window.localStorage.setItem(WORKSPACE_ENTITY_KEY, entityId);
		}
	}

	setPeriodMonth(periodMonth: string): void {
		if (/^\d{4}-\d{2}$/.test(periodMonth)) {
			this.selectedPeriodMonth = periodMonth;
		}
	}

	setEntities(entities: LegalEntityDTO[]): void {
		this.entities = entities;
		if (
			this.selectedEntityId !== 'ALL' &&
			entities.length > 0 &&
			!entities.some((entity) => entity.id === this.selectedEntityId)
		) {
			this.setEntityId('ALL');
		}
	}
}

export const workspaceStore = new WorkspaceStore();
