import { describe, expect, it } from 'vitest';
import {
	formatBasisPointsToPercent,
	formatCentsToEuro,
	formatCentsToEuroInput,
	parseEuroToCents,
	parsePercentToBasisPoints
} from './money';

describe('money and basis points boundary formatters', () => {
	it('formats integer cents to French euro strings without float drift', () => {
		expect(formatCentsToEuro(0)).toContain('0,00');
		expect(formatCentsToEuro(92050)).toMatch(/920,50\s*€/);
		expect(formatCentsToEuro(-142000)).toMatch(/-1[\s\u202f\u00a0]?420,00\s*€/);
	});

	it('formats integer cents into editable French euro input strings without float division', () => {
		expect(formatCentsToEuroInput(0)).toBe('0,00');
		expect(formatCentsToEuroInput(123456)).toBe('1234,56');
		expect(formatCentsToEuroInput(5)).toBe('0,05');
		expect(formatCentsToEuroInput(-505)).toBe('-5,05');
	});

	it('parses French and dot decimal euro inputs into exact integer cents', () => {
		expect(parseEuroToCents('1 234,56 €')).toBe(123456);
		expect(parseEuroToCents('920,5')).toBe(92050);
		expect(parseEuroToCents('580')).toBe(58000);
		expect(() => parseEuroToCents('-10')).toThrow(/invalid monetary amount/i);
		expect(() => parseEuroToCents('abc')).toThrow(/invalid monetary amount/i);
	});

	it('formats and parses basis points (10000 = 100.00%)', () => {
		expect(formatBasisPointsToPercent(10000)).toMatch(/100,00\s*%/);
		expect(formatBasisPointsToPercent(2000)).toMatch(/20,00\s*%/);
		expect(parsePercentToBasisPoints('20,5 %')).toBe(2050);
	});
});
