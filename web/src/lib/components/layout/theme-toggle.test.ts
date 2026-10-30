import { describe, expect, it, vi } from 'vitest';
import { applyThemePreference, formatSystemThemeLabel } from './theme-toggle';

describe('ThemeToggle helpers', () => {
	it('dispatches setMode for light/dark and resetMode for system preference', () => {
		const setMode = vi.fn();
		const resetMode = vi.fn();

		applyThemePreference('light', { setMode, resetMode });
		expect(setMode).toHaveBeenCalledWith('light');
		expect(resetMode).not.toHaveBeenCalled();

		applyThemePreference('dark', { setMode, resetMode });
		expect(setMode).toHaveBeenCalledWith('dark');

		applyThemePreference('system', { setMode, resetMode });
		expect(resetMode).toHaveBeenCalledTimes(1);

		applyThemePreference('invalid', { setMode, resetMode });
		expect(setMode).toHaveBeenCalledTimes(2);
		expect(resetMode).toHaveBeenCalledTimes(1);
	});

	it('formats the system theme label with current OS scheme hint when available', () => {
		expect(formatSystemThemeLabel('light')).toBe('Système (clair)');
		expect(formatSystemThemeLabel('dark')).toBe('Système (sombre)');
		expect(formatSystemThemeLabel(undefined)).toBe('Système');
	});
});
