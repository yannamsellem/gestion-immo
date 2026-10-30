import type { SystemModeValue } from 'mode-watcher';

export type ThemePreference = 'light' | 'dark' | 'system';

export function applyThemePreference(
	value: string,
	actions: {
		setMode: (mode: 'light' | 'dark') => void;
		resetMode: () => void;
	}
): void {
	if (value === 'light' || value === 'dark') {
		actions.setMode(value);
		return;
	}
	if (value === 'system') {
		actions.resetMode();
	}
}

export function formatSystemThemeLabel(systemMode: SystemModeValue): string {
	if (systemMode === 'light') {
		return 'Système (clair)';
	}
	if (systemMode === 'dark') {
		return 'Système (sombre)';
	}
	return 'Système';
}
