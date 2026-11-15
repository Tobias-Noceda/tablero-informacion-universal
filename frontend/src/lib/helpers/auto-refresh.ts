// How long a card waits before fetching again: a random span between its
// refresh rate and a minute more, so cards (and viewers) on a board spread
// their requests instead of firing together. Never under MIN_DELAY, so a
// rate-0 card ("fetch every time") does not hammer its API.
export const SPREAD_SECONDS = 60;
export const MIN_DELAY_SECONDS = 30;

// `minRate` is the board plan's floor (limits.min_rate): the effective
// interval is max(card rate, minRate), as on the backend's cache.
export function nextDelay(rate: number, minRate = 0, random: () => number = Math.random): number {
	const card = Number.isFinite(rate) && rate > 0 ? rate : 0;
	const floor = Number.isFinite(minRate) && minRate > 0 ? minRate : 0;
	const base = Math.max(card, floor);
	const seconds = Math.max(MIN_DELAY_SECONDS, base + random() * SPREAD_SECONDS);
	return Math.round(seconds * 1000);
}
