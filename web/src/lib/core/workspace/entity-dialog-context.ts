import { getContext, setContext } from 'svelte';

export interface EntityDialogContextApi {
	openCreateDialog: () => void;
}

const ENTITY_DIALOG_CONTEXT_KEY = Symbol('entity-dialog-context');

export function setEntityDialogContext(api: EntityDialogContextApi): void {
	setContext(ENTITY_DIALOG_CONTEXT_KEY, api);
}

export function useEntityDialogContext(): EntityDialogContextApi {
	const context = getContext<EntityDialogContextApi | undefined>(ENTITY_DIALOG_CONTEXT_KEY);
	if (!context) {
		throw new Error('useEntityDialogContext must be called within (app)/+layout.svelte');
	}
	return context;
}
