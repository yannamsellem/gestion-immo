const dateFormatterFr = new Intl.DateTimeFormat('fr-FR', {
	day: '2-digit',
	month: 'short',
	year: 'numeric'
});

const monthFormatterFr = new Intl.DateTimeFormat('fr-FR', {
	month: 'long',
	year: 'numeric'
});

export function formatIsoDateFr(isoDate: string | null | undefined): string {
	if (!isoDate) {
		return '—';
	}
	const parsed = new Date(isoDate);
	if (Number.isNaN(parsed.getTime())) {
		return isoDate;
	}
	return dateFormatterFr.format(parsed);
}

export function formatPeriodMonthFr(periodMonth: string): string {
	const match = /^(\d{4})-(\d{2})$/.exec(periodMonth.trim());
	if (!match) {
		return periodMonth;
	}
	const year = Number.parseInt(match[1], 10);
	const monthIndex = Number.parseInt(match[2], 10) - 1;
	const date = new Date(Date.UTC(year, monthIndex, 1));
	const formatted = monthFormatterFr.format(date);
	return formatted.charAt(0).toUpperCase() + formatted.slice(1);
}

export function getCurrentPeriodMonth(now: Date = new Date()): string {
	const year = now.getFullYear();
	const month = String(now.getMonth() + 1).padStart(2, '0');
	return `${year}-${month}`;
}
