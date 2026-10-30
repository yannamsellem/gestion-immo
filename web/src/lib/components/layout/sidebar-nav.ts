function normalizePathname(pathname: string): string {
	const trimmed = pathname.trim();
	if (!trimmed || trimmed === '/') {
		return '/';
	}
	const stripped = trimmed.replace(/\/+$/, '');
	return stripped || '/';
}

export function isExactRouteActive(targetPath: string, currentPathname: string): boolean {
	return normalizePathname(targetPath) === normalizePathname(currentPathname);
}

export function isGroupPathActive(basePath: string, currentPathname: string): boolean {
	const normalizedBase = normalizePathname(basePath);
	const normalizedCurrent = normalizePathname(currentPathname);

	if (normalizedBase === '/') {
		return normalizedCurrent === '/';
	}

	return normalizedCurrent === normalizedBase || normalizedCurrent.startsWith(`${normalizedBase}/`);
}

export function formatUserInitials(fullName: string | null | undefined): string {
	const trimmed = (fullName ?? '').trim();
	if (!trimmed) {
		return 'GI';
	}

	const words = trimmed.split(/\s+/).filter(Boolean);
	if (words.length >= 2) {
		return `${words[0][0]}${words[words.length - 1][0]}`.toUpperCase();
	}

	return words[0].slice(0, 2).toUpperCase();
}
