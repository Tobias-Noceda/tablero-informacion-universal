import { m } from '$lib/paraglide/messages';
import { getLocale } from '$lib/paraglide/runtime';
import type { Plan } from '$types/api';

const names: Record<Plan, () => string> = {
	free: m['billing.plan.free'],
	pro: m['billing.plan.pro'],
	max: m['billing.plan.max']
};

export function planName(plan: Plan) {
	return (names[plan] ?? names.free)();
}

// A whole amount of a currency, written the way the UI language does.
export function money(amount: number, currency: string) {
	return new Intl.NumberFormat(getLocale(), {
		style: 'currency',
		currency,
		maximumFractionDigits: 0
	}).format(amount);
}

// A refresh interval in whole minutes and seconds ("10 min", "2 min 30 s").
export function interval(seconds: number) {
	const minutes = Math.floor(seconds / 60);
	const rest = seconds % 60;
	if (minutes === 0) return m['billing.interval.seconds']({ seconds: rest });
	if (rest === 0) return m['billing.interval.minutes']({ minutes });
	return m['billing.interval.minutes_seconds']({ minutes, seconds: rest });
}

export function day(iso: string) {
	return new Date(iso).toLocaleDateString(getLocale(), { dateStyle: 'long' });
}
