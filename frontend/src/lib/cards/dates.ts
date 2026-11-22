// Date and date-time parameters: what the card stores (and its API reads)
// against what the browser's pickers show. A date is YYYY-MM-DD in both. A
// date-time is stored in RFC 3339, in UTC (2026-01-31T12:00:00Z), and shown in
// the person's own time zone, which is what `datetime-local` takes.

export type DateKind = 'date' | 'datetime';

const DATE = /^\d{4}-\d{2}-\d{2}$/;

const pad = (n: number) => String(n).padStart(2, '0');

/** The picker's value for what the card stores; '' when it cannot be read. */
export function toPicker(kind: DateKind, stored: string | undefined): string {
	if (!stored) return '';
	if (kind === 'date') return DATE.test(stored) ? stored : '';
	const instant = new Date(stored);
	if (Number.isNaN(instant.getTime())) return '';
	return (
		`${instant.getFullYear()}-${pad(instant.getMonth() + 1)}-${pad(instant.getDate())}` +
		`T${pad(instant.getHours())}:${pad(instant.getMinutes())}`
	);
}

/** What the card stores for the picker's value; '' when it is empty. */
export function fromPicker(kind: DateKind, picked: string): string {
	if (!picked) return '';
	if (kind === 'date') return picked;
	const instant = new Date(picked);
	if (Number.isNaN(instant.getTime())) return '';
	return instant.toISOString().replace(/\.\d{3}Z$/, 'Z');
}

/** Whether a stored value is one the backend takes for that kind. */
export function validDate(kind: DateKind, stored: string): boolean {
	if (kind === 'date') {
		if (!DATE.test(stored)) return false;
		const [year, month, dayOfMonth] = stored.split('-').map(Number);
		const day = new Date(Date.UTC(year, month - 1, dayOfMonth));
		return (
			day.getUTCFullYear() === year &&
			day.getUTCMonth() === month - 1 &&
			day.getUTCDate() === dayOfMonth
		);
	}
	return (
		/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})$/.test(stored) &&
		!Number.isNaN(new Date(stored).getTime())
	);
}

export const isDateKind = (type: string): type is DateKind =>
	type === 'date' || type === 'datetime';
