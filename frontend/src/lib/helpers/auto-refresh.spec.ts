import { describe, expect, it } from 'vitest';
import { MIN_DELAY_SECONDS, nextDelay, SPREAD_SECONDS } from './auto-refresh';

describe('nextDelay', () => {
	it('spans from the rate to a minute more', () => {
		expect(nextDelay(600, 0, () => 0)).toBe(600_000);
		expect(nextDelay(600, 0, () => 0.5)).toBe(630_000);
		expect(nextDelay(600, 0, () => 0.999999)).toBeLessThanOrEqual((600 + SPREAD_SECONDS) * 1000);
	});

	it('never goes under the floor', () => {
		expect(nextDelay(0, 0, () => 0)).toBe(MIN_DELAY_SECONDS * 1000);
		expect(nextDelay(-5, 0, () => 0)).toBe(MIN_DELAY_SECONDS * 1000);
		expect(nextDelay(Number.NaN, 0, () => 1)).toBe(SPREAD_SECONDS * 1000);
	});

	it('uses the plan interval when the card asks for less', () => {
		expect(nextDelay(5, 300, () => 0)).toBe(300_000);
		expect(nextDelay(0, 150, () => 0)).toBe(150_000);
		expect(nextDelay(5, 60, () => 0)).toBe(60_000);
	});

	it('keeps the card rate when it is above the plan interval', () => {
		expect(nextDelay(900, 300, () => 0)).toBe(900_000);
	});
});
