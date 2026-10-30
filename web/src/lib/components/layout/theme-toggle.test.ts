import { readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it, vi } from 'vitest';
import { applyThemePreference, formatSystemThemeLabel } from './theme-toggle';

const currentDir = dirname(fileURLToPath(import.meta.url));
const themeToggleSveltePath = resolve(currentDir, './ThemeToggle.svelte');

describe('ThemeToggle behavior & DropdownMenu RadioItem contract', () => {
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

	it('composes DropdownMenu.Label, DropdownMenu.RadioGroup, DropdownMenu.RadioItem, and Sun/Moon/DeviceDesktop Tabler icons', () => {
		const source = readFileSync(themeToggleSveltePath, 'utf8');

		expect(source).toContain('@tabler/icons-svelte/icons/sun');
		expect(source).toContain('@tabler/icons-svelte/icons/moon');
		expect(source).toContain('@tabler/icons-svelte/icons/device-desktop');
		expect(source).toContain("mode.current === 'light'");
		expect(source).toContain('<DropdownMenu.Label');
		expect(source).toContain('<DropdownMenu.RadioGroup');
		expect(source).toContain('<DropdownMenu.RadioItem');
		expect(source.match(/<DropdownMenu\.RadioItem[^>]*\bcloseOnSelect\b/g)).toHaveLength(3);
		expect(source).not.toContain('<DropdownMenu.Item');
	});
});
