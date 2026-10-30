import { readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it, vi } from 'vitest';

const { invalidateAllMock } = vi.hoisted(() => ({
	invalidateAllMock: vi.fn<() => Promise<void>>(() => Promise.resolve())
}));

vi.mock('$app/navigation', () => ({
	invalidateAll: invalidateAllMock
}));

import { deriveWorkspaceGateState } from './onboarding-gate';
import { workspaceStore } from './workspace.svelte';

const currentDir = dirname(fileURLToPath(import.meta.url));
const createEntityDialogPath = resolve(
	currentDir,
	'../../components/portfolio/CreateEntityDialog.svelte'
);
const appLayoutPath = resolve(currentDir, '../../../routes/(app)/+layout.svelte');

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

describe('persistent onboarding accessibility & async completion contract', () => {
	it('exposes ThemeToggle and single logout action inside persistent CreateEntityDialog without focus-trapped outer controls', () => {
		const dialogSource = readFileSync(createEntityDialogPath, 'utf8');
		const layoutSource = readFileSync(appLayoutPath, 'utf8');

		expect(dialogSource).toContain('<ThemeToggle');
		expect(dialogSource).toContain('/(auth)/logout');

		const onboardingBranch =
			layoutSource
				.split("{:else if gateState === 'onboarding_required'}")[1]
				?.split('{:else}')[0] ?? '';
		expect(onboardingBranch).toContain('<CreateEntityDialog open={true} persistent={true} />');
		expect(onboardingBranch).not.toContain('<ThemeToggle');
		expect(onboardingBranch).not.toContain('/(auth)/logout');
	});

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
			const resultPromise = Promise.resolve(workspaceStore.setEntityId('ent_new')).then(() => {
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
