import { describe, expect, it } from 'vitest';
import { formatUserInitials, isExactRouteActive, isGroupPathActive } from './sidebar-nav';

describe('isGroupPathActive', () => {
	it('matches exact group root and nested sub-routes without matching sibling prefixes', () => {
		expect(isGroupPathActive('/leases', '/leases')).toBe(true);
		expect(isGroupPathActive('/leases', '/leases/tenants')).toBe(true);
		expect(isGroupPathActive('/leases', '/leases/templates/')).toBe(true);
		expect(isGroupPathActive('/leases', '/leases-other')).toBe(false);
		expect(isGroupPathActive('/', '/')).toBe(true);
		expect(isGroupPathActive('/', '/leases')).toBe(false);
	});
});

describe('isExactRouteActive', () => {
	it('matches only exact normalized route pathnames so parent and child sub-items do not double-highlight', () => {
		expect(isExactRouteActive('/leases', '/leases')).toBe(true);
		expect(isExactRouteActive('/leases', '/leases/tenants')).toBe(false);
		expect(isExactRouteActive('/leases/tenants', '/leases/tenants/')).toBe(true);
		expect(isExactRouteActive('/', '/')).toBe(true);
		expect(isExactRouteActive('/', '/properties')).toBe(false);
	});
});

describe('formatUserInitials', () => {
	it('derives two-letter uppercase initials with GI fallback', () => {
		expect(formatUserInitials('Yann Amsellem')).toBe('YA');
		expect(formatUserInitials('Jean-Pierre Martin')).toBe('JM');
		expect(formatUserInitials('Admin')).toBe('AD');
		expect(formatUserInitials('a')).toBe('A');
		expect(formatUserInitials('')).toBe('GI');
		expect(formatUserInitials(null)).toBe('GI');
		expect(formatUserInitials(undefined)).toBe('GI');
	});
});
