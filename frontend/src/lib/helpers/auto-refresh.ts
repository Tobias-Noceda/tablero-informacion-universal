// How long a card waits before fetching again: a random span between its
// refresh rate and a minute more, so cards (and viewers) on a board spread
// their requests instead of firing together. Never under MIN_DELAY, so a
// rate-0 card ("fetch every time") does not hammer its API.
export const SPREAD_SECONDS = 60;
export const MIN_DELAY_SECONDS = 30;

export function nextDelay(rate: number, random: () => number = Math.random): number {
	const base = Number.isFinite(rate) && rate > 0 ? rate : 0;
	const seconds = Math.max(MIN_DELAY_SECONDS, base + random() * SPREAD_SECONDS);
	return Math.round(seconds * 1000);
}
