export type WorkspaceGateState = 'onboarding_required' | 'load_error' | 'ready';

export function deriveWorkspaceGateState(params: {
	entitiesCount: number;
	entitiesLoadError: string | null;
}): WorkspaceGateState {
	if (params.entitiesCount > 0) {
		return 'ready';
	}
	if (params.entitiesLoadError !== null) {
		return 'load_error';
	}
	return 'onboarding_required';
}
