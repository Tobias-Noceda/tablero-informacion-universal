import { describe, expect, it } from 'vitest';

import { fromPicker, isDateKind, toPicker, validDate } from './dates';

describe('date parameters', () => {
	it('keep YYYY-MM-DD as is', () => {
		expect(toPicker('date', '2026-11-22')).toBe('2026-11-22');
		expect(fromPicker('date', '2026-11-22')).toBe('2026-11-22');
	});

	it('show nothing for a stored value that is not a date', () => {
		expect(toPicker('date', '22/11/2026')).toBe('');
		expect(toPicker('date', undefined)).toBe('');
		expect(fromPicker('date', '')).toBe('');
	});
});

describe('date-time parameters', () => {
	it('go from the local picker to UTC and back', () => {
		const stored = fromPicker('datetime', '2026-11-22T09:30');
		expect(stored).toMatch(/^2026-11-2[12]T\d{2}:30:00Z$/);
		expect(new Date(stored).getTime()).toBe(new Date('2026-11-22T09:30').getTime());
		expect(toPicker('datetime', stored)).toBe('2026-11-22T09:30');
	});

	it('read a stored instant with an offset', () => {
		const shown = toPicker('datetime', '2026-01-01T00:00:00-03:00');
		expect(new Date(shown).getTime()).toBe(new Date('2026-01-01T03:00:00Z').getTime());
	});

	it('show nothing for what is not an instant', () => {
		expect(toPicker('datetime', 'tomorrow')).toBe('');
		expect(fromPicker('datetime', '')).toBe('');
	});
});

describe('validDate', () => {
	it('takes what the backend takes', () => {
		expect(validDate('date', '2026-02-28')).toBe(true);
		expect(validDate('date', '2026-02-30')).toBe(false);
		expect(validDate('date', '2026-2-3')).toBe(false);
		expect(validDate('datetime', '2026-01-01T00:00:00Z')).toBe(true);
		expect(validDate('datetime', '2026-01-01T00:00:00-03:00')).toBe(true);
		expect(validDate('datetime', '2026-01-01T00:00')).toBe(false);
		expect(validDate('datetime', '2026-01-01')).toBe(false);
	});
});

describe('isDateKind', () => {
	it('knows the two date types', () => {
		expect(isDateKind('date')).toBe(true);
		expect(isDateKind('datetime')).toBe(true);
		expect(isDateKind('string')).toBe(false);
	});
});
