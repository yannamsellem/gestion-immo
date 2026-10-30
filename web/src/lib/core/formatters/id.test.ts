import { describe, expect, it } from 'vitest';
import { formatShortId } from './id';

describe('formatShortId', () => {
	it('extracts trailing random hex digits so same-millisecond UUIDv7 values do not collide', () => {
		const idA = '0192dce0-7c4a-7001-8abc-111122223333';
		const idB = '0192dce0-7c4a-7002-9def-444455556666';

		expect(formatShortId(idA)).toBe('22223333');
		expect(formatShortId(idB)).toBe('55556666');
		expect(formatShortId(idA, 6)).toBe('223333');
		expect(formatShortId(idB, 6)).toBe('556666');
		expect(formatShortId(idA, 6)).not.toBe(formatShortId(idB, 6));
	});

	it('strips hyphens and handles short or empty strings safely', () => {
		expect(formatShortId('  abc-12  ', 8)).toBe('abc12');
		expect(formatShortId('', 8)).toBe('');
	});
});
