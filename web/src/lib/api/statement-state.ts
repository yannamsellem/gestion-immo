// web/src/lib/api/statement-state.ts

/**
 * Immutably toggles a financial statement line ID in the disabled set.
 * Returns a brand-new ReadonlySet<string> without mutating the input set.
 */
export function toggleStatementLine(
	disabledIds: ReadonlySet<string>,
	lineId: string
): ReadonlySet<string> {
	const normalized = lineId.trim();
	const next = new Set<string>(disabledIds);
	if (!normalized) {
		return next;
	}
	if (next.has(normalized)) {
		next.delete(normalized);
	} else {
		next.add(normalized);
	}
	return next;
}

/**
 * Returns a deterministically sorted array of disabled line IDs suitable for
 * POST /api/v1/entities/{id}/statements/preview and /pdf payloads.
 */
export function toDisabledArray(disabledIds: ReadonlySet<string>): string[] {
	return Array.from(disabledIds)
		.map((id) => id.trim())
		.filter((id) => id.length > 0)
		.sort((a, b) => a.localeCompare(b));
}

/**
 * Checks whether a given line ID or account code is currently disabled.
 */
export function isLineDisabled(disabledIds: ReadonlySet<string>, lineId: string): boolean {
	return disabledIds.has(lineId.trim());
}
