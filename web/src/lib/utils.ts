// web/src/lib/utils.ts

export type ClassValue =
	| string
	| number
	| boolean
	| null
	| undefined
	| ClassValue[]
	| Record<string, boolean | null | undefined>;

function toVal(mix: ClassValue): string {
	if (typeof mix === 'string' || typeof mix === 'number') {
		return String(mix);
	}
	if (!mix || typeof mix !== 'object') {
		return '';
	}
	if (Array.isArray(mix)) {
		return mix.map(toVal).filter(Boolean).join(' ');
	}
	return Object.entries(mix)
		.filter(([, enabled]) => Boolean(enabled))
		.map(([cls]) => cls)
		.join(' ');
}

export function cn(...inputs: ClassValue[]): string {
	return inputs.map(toVal).filter(Boolean).join(' ');
}

const eurFormatter = new Intl.NumberFormat('fr-FR', {
	style: 'currency',
	currency: 'EUR',
	minimumFractionDigits: 2,
	maximumFractionDigits: 2
});

const pctFormatter = new Intl.NumberFormat('fr-FR', {
	minimumFractionDigits: 2,
	maximumFractionDigits: 2
});

/**
 * Formats an integer amount in euro cents into French EUR currency notation.
 */
export function formatEUR(cents: number): string {
	return eurFormatter.format(cents / 100);
}

/**
 * Formats integer basis points (10000 = 100,00 %) into French percentage notation.
 */
export function formatBasisPointsPct(basisPoints: number, notSignificant = false): string {
	if (notSignificant) {
		return 'N/S';
	}
	return `${pctFormatter.format(basisPoints / 100)} %`;
}

/**
 * Computes and formats the N vs N-1 percentage variation from integer cents,
 * returning 'N/S' when netNMinus1Cents is 0.
 */
export function formatVariationPct(netNCents: number, netNMinus1Cents: number): string {
	if (netNMinus1Cents === 0) {
		return 'N/S';
	}
	const diff = netNCents - netNMinus1Cents;
	const base = Math.abs(netNMinus1Cents);
	const bps =
		diff >= 0
			? Math.floor((diff * 10000 + Math.floor(base / 2)) / base)
			: -Math.floor((-diff * 10000 + Math.floor(base / 2)) / base);
	return formatBasisPointsPct(bps, false);
}
