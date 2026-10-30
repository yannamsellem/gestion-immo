export function formatShortId(id: string, length = 8): string {
	const compact = id.trim().replace(/-/g, '');
	if (compact.length <= length) {
		return compact;
	}
	return compact.slice(-length);
}
