const euroFormatter = new Intl.NumberFormat('fr-FR', {
	style: 'currency',
	currency: 'EUR',
	minimumFractionDigits: 2,
	maximumFractionDigits: 2
});

const percentFormatter = new Intl.NumberFormat('fr-FR', {
	minimumFractionDigits: 2,
	maximumFractionDigits: 2
});

/**
 * Formats integer euro cents (int64 equivalent) into a human-readable French euro string (e.g. "920,50 €").
 * Uses integer division and remainder decomposition before formatting to avoid binary float drift.
 */
export function formatCentsToEuro(cents: number): string {
	if (!Number.isFinite(cents)) {
		return euroFormatter.format(0);
	}
	const roundedCents = Math.trunc(cents);
	return euroFormatter.format(roundedCents / 100);
}

/**
 * Parses a user-entered French or dot-decimal euro string (e.g. "1 234,56 €", "920,5", "580")
 * into exact integer cents without floating-point multiplication.
 */
export function parseEuroToCents(raw: string, options?: { allowNegative?: boolean }): number {
	const cleaned = raw
		.replace(/€/g, '')
		.replace(/[\s\u00a0\u202f]/g, '')
		.replace(',', '.')
		.trim();

	if (!cleaned) {
		throw new Error(`Invalid monetary amount: "${raw}"`);
	}

	const match = /^([+-])?(\d+)(?:\.(\d{1,2}))?$/.exec(cleaned);
	if (!match) {
		throw new Error(`Invalid monetary amount: "${raw}"`);
	}

	const sign = match[1] === '-' ? -1 : 1;
	if (sign < 0 && !options?.allowNegative) {
		throw new Error(`Invalid monetary amount (negative not allowed): "${raw}"`);
	}

	const wholeUnits = Number.parseInt(match[2], 10);
	const fractionStr = (match[3] ?? '').padEnd(2, '0');
	const fractionCents = Number.parseInt(fractionStr, 10);

	const totalCents = wholeUnits * 100 + fractionCents;
	return sign < 0 ? -totalCents : totalCents;
}

/**
 * Formats integer basis points (10000 = 100.00%) into a human-readable French percentage string ("20,00 %").
 */
export function formatBasisPointsToPercent(basisPoints: number): string {
	if (!Number.isFinite(basisPoints)) {
		return '0,00 %';
	}
	const roundedBp = Math.trunc(basisPoints);
	return `${percentFormatter.format(roundedBp / 100)} %`;
}

/**
 * Parses a percentage string (e.g. "20,5 %" or "100") into integer basis points (where 100% = 10000).
 */
export function parsePercentToBasisPoints(raw: string): number {
	const cleaned = raw
		.replace(/%/g, '')
		.replace(/[\s\u00a0\u202f]/g, '')
		.replace(',', '.')
		.trim();

	const match = /^(\d+)(?:\.(\d{1,2}))?$/.exec(cleaned);
	if (!match) {
		throw new Error(`Invalid percentage value: "${raw}"`);
	}

	const wholePercent = Number.parseInt(match[1], 10);
	const fractionStr = (match[2] ?? '').padEnd(2, '0');
	const fractionBp = Number.parseInt(fractionStr, 10);

	const totalBp = wholePercent * 100 + fractionBp;
	if (totalBp < 0 || totalBp > 10000) {
		throw new Error(`Percentage out of bounds (0% - 100%): "${raw}"`);
	}
	return totalBp;
}
