import { page } from 'vitest/browser';
import { describe, expect, it } from 'vitest';
import { render } from 'vitest-browser-svelte';

import DateInput from './DateInput.svelte';

describe('DateInput.svelte', () => {
	it('stores a picked day as YYYY-MM-DD', async () => {
		const props = $state({ kind: 'date' as const, value: '', label: 'Start date' });
		render(DateInput, props);

		await page.getByLabelText('Start date').fill('2026-11-22');
		expect(props.value).toBe('2026-11-22');
	});

	it('stores a picked local time in UTC and says so', async () => {
		const props = $state({ kind: 'datetime' as const, value: '', label: 'From' });
		render(DateInput, props);

		await page.getByLabelText('From').fill('2026-11-22T09:30');
		expect(new Date(props.value).getTime()).toBe(new Date('2026-11-22T09:30').getTime());
		expect(props.value).toMatch(/Z$/);
		await expect.element(page.getByText(props.value, { exact: false })).toBeInTheDocument();
	});

	it('shows a stored instant in local time', async () => {
		const stored = new Date('2026-01-01T12:00').toISOString().replace(/\.\d{3}Z$/, 'Z');
		render(DateInput, { kind: 'datetime', value: stored, label: 'From' });

		await expect.element(page.getByLabelText('From')).toHaveValue('2026-01-01T12:00');
	});

	it('flags a value saved as free text before', async () => {
		render(DateInput, { kind: 'date', value: 'next monday', label: 'Start date' });

		await expect.element(page.getByText('next monday', { exact: false })).toBeInTheDocument();
	});
});
