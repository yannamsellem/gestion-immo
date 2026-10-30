import { describe, expect, it } from 'vitest';
import { deriveWorkspaceGateState } from './onboarding-gate';

describe('deriveWorkspaceGateState', () => {
	it('returns onboarding_required when zero entities exist and there is no load error', () => {
		expect(
			deriveWorkspaceGateState({
				entitiesCount: 0,
				entitiesLoadError: null
			})
		).toBe('onboarding_required');
	});

	it('returns load_error when zero entities exist due to a backend load error (never locks user into onboarding modal)', () => {
		expect(
			deriveWorkspaceGateState({
				entitiesCount: 0,
				entitiesLoadError: 'Backend 502'
			})
		).toBe('load_error');
	});

	it('returns ready when one or more entities exist with no load error', () => {
		expect(
			deriveWorkspaceGateState({
				entitiesCount: 2,
				entitiesLoadError: null
			})
		).toBe('ready');
	});

	it('returns ready when at least one entity is available even if a stale load warning is present', () => {
		expect(
			deriveWorkspaceGateState({
				entitiesCount: 1,
				entitiesLoadError: 'stale warning'
			})
		).toBe('ready');
	});
});
