import { describe, expect, it } from 'vitest';
import { formatUpdated } from './freshness';

const now = Date.parse('2026-11-21T14:30:00');

describe('formatUpdated', () => {
	it('gives only the time for data from today', () => {
		const text = formatUpdated(Date.parse('2026-11-21T09:05:00'), now, 'es');
		expect(text).toMatch(/09:05/);
		expect(text).not.toMatch(/nov/i);
	});

	it('adds the day for older data', () => {
		const text = formatUpdated(Date.parse('2026-11-20T23:50:00'), now, 'en');
		expect(text).toMatch(/Nov/);
		expect(text).toMatch(/20/);
	});
});
