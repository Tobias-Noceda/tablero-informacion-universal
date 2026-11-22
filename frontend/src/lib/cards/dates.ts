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
	const at = new Date(stored);
	if (Number.isNaN(at.getTime())) return '';
	return (
		`${at.getFullYear()}-${pad(at.getMonth() + 1)}-${pad(at.getDate())}` +
		`T${pad(at.getHours())}:${pad(at.getMinutes())}`
	);
}

/** What the card stores for the picker's value; '' when it is empty. */
export function fromPicker(kind: DateKind, picked: string): string {
	if (!picked) return '';
	if (kind === 'date') return picked;
	const at = new Date(picked);
	if (Number.isNaN(at.getTime())) return '';
	return at.toISOString().replace(/\.\d{3}Z$/, 'Z');
}

/** Whether a stored value is one the backend takes for that kind. */
export function validDate(kind: DateKind, stored: string): boolean {
	if (kind === 'date') {
		if (!DATE.test(stored)) return false;
		const [y, m, d] = stored.split('-').map(Number);
		const day = new Date(Date.UTC(y, m - 1, d));
		return day.getUTCFullYear() === y && day.getUTCMonth() === m - 1 && day.getUTCDate() === d;
	}
	return (
		/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})$/.test(stored) &&
		!Number.isNaN(new Date(stored).getTime())
	);
}

export const isDateKind = (type: string): type is DateKind =>
	type === 'date' || type === 'datetime';
