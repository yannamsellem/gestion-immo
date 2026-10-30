import { describe, expect, it, vi } from 'vitest';

const { invalidateAllMock } = vi.hoisted(() => ({
	invalidateAllMock: vi.fn<() => Promise<void>>(() => Promise.resolve())
}));

vi.mock('$app/navigation', () => ({
	invalidateAll: invalidateAllMock
}));

import { deriveWorkspaceGateState } from './onboarding-gate';
import { workspaceStore } from './workspace.svelte';

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

describe('workspaceStore.setEntityId async completion contract', () => {
	it('awaits invalidateAll inside workspaceStore.setEntityId before resolving', async () => {
		let resolveInvalidate: (() => void) | null = null;
		invalidateAllMock.mockImplementationOnce(
			() =>
				new Promise<void>((res) => {
					resolveInvalidate = res;
				})
		);

		const previousDocument = globalThis.document;
		Object.defineProperty(globalThis, 'document', {
			value: { cookie: '' },
			configurable: true,
			writable: true
		});

		try {
			let settled = false;
			const resultPromise = Promise.resolve(
				workspaceStore.setEntityId('0192dce0-7c4a-7003-8000-000000000001')
			).then(() => {
				settled = true;
			});

			await Promise.resolve();
			expect(invalidateAllMock).toHaveBeenCalledTimes(1);
			expect(settled).toBe(false);

			resolveInvalidate!();
			await resultPromise;
			expect(settled).toBe(true);
		} finally {
			Object.defineProperty(globalThis, 'document', {
				value: previousDocument,
				configurable: true,
				writable: true
			});
		}
	});
});
