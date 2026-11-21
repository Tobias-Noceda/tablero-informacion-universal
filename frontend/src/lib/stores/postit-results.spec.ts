import { beforeEach, describe, expect, it, vi } from 'vitest';

const api = vi.hoisted(() => {
	class CardError extends Error {
		constructor(
			readonly status: number,
			readonly code?: string,
			readonly sourceStatus?: number
		) {
			super(code);
		}
	}
	return {
		CardError,
		execute: vi.fn(),
		get_settings: vi.fn(async () => ({ resource: 'https://example.com', rate: 60 }))
	};
});
vi.mock('$services/post-it', () => api);
vi.mock('$stores/plan.svelte', () => ({ boardPlan: { limits: { min_rate: 0 } } }));
vi.mock('$modules/session.svelte', () => ({ session: { onUserChange: () => {} } }));

const { results } = await import('./postit-results.svelte');

const sourceDown = () => new api.CardError(504, 'source_timeout');

describe('card freshness', () => {
	beforeEach(() => {
		results.clear();
		api.execute.mockReset();
		api.get_settings.mockClear();
		vi.useFakeTimers();
		vi.setSystemTime(new Date('2026-11-21T14:30:00'));
	});

	it('records when the shown data arrived', async () => {
		api.execute.mockResolvedValueOnce({ value: '1' });

		await results.get('card');

		expect(results.freshness('card')).toEqual({ at: Date.parse('2026-11-21T14:30:00') });
	});

	it('keeps the data on screen when a refresh fails, and says so', async () => {
		results.watch('card');
		api.execute.mockResolvedValueOnce({ value: '1' });
		await results.get('card');
		vi.setSystemTime(new Date('2026-11-21T14:35:00'));

		api.execute.mockRejectedValueOnce(sourceDown());
		await results.revalidate('card');

		await expect(results.get('card')).resolves.toEqual({ value: '1' });
		expect(results.freshness('card')).toEqual({
			at: Date.parse('2026-11-21T14:30:00'),
			failure: { kind: 'timeout' }
		});
	});

	it('clears the warning once a refresh works again', async () => {
		results.watch('card');
		api.execute.mockResolvedValueOnce({ value: '1' });
		await results.get('card');
		api.execute.mockRejectedValueOnce(sourceDown());
		await results.revalidate('card');
		vi.setSystemTime(new Date('2026-11-21T14:40:00'));

		api.execute.mockResolvedValueOnce({ value: '2' });
		await results.revalidate('card');

		await expect(results.get('card')).resolves.toEqual({ value: '2' });
		expect(results.freshness('card')).toEqual({ at: Date.parse('2026-11-21T14:40:00') });
	});

	it('has no time to show when the card never answered', async () => {
		api.execute.mockRejectedValueOnce(new api.CardError(502, 'source_status', 404));

		await expect(results.get('card')).rejects.toThrow();
		await Promise.resolve();

		expect(results.freshness('card')).toEqual({ failure: { kind: 'not_found' } });
	});

	it('knows which cards fetch anything once their settings are read', async () => {
		api.get_settings.mockResolvedValueOnce({ resource: '', rate: 0 });
		results.watch('static');
		results.watch('live');
		await vi.waitFor(() => expect(api.get_settings).toHaveBeenCalledTimes(2));
		await Promise.resolve();

		expect(results.polls('static')).toBe(false);
		expect(results.polls('live')).toBe(true);
	});

	it('forgets a card that is dropped', async () => {
		api.execute.mockResolvedValueOnce({ value: '1' });
		await results.get('card');

		results.drop('card');

		expect(results.freshness('card')).toBeUndefined();
	});
});
