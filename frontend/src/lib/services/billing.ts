import * as api from '$modules/api.svelte';
import type { Plan, PlanCatalogue, Subscription, Usage } from '$types/api';

// Why a billing call failed, as the backend's `error` code (invalid_plan,
// already_subscribed, billing_disabled...).
export class BillingError extends Error {
	constructor(readonly code: string) {
		super(code);
		this.name = 'BillingError';
	}
}

async function ensure(response: Response) {
	if (response.ok) return response;
	const body = await response.json().catch(() => ({}));
	throw new BillingError(typeof body?.error === 'string' ? body.error : 'unknown');
}

// The three plans with their price and limits. Public: it works signed out.
export async function plans() {
	const response = await ensure(await api.request('GET', '/v1/billing/plans'));
	return (await response.json()) as PlanCatalogue;
}

// The caller's subscription; `status: "none"` when they never subscribed.
export async function subscription() {
	const response = await ensure(await api.request('GET', '/v1/billing/subscription'));
	return (await response.json()) as Subscription;
}

// How much of their own limits the caller uses.
export async function usage() {
	const response = await ensure(await api.request('GET', '/v1/billing/usage'));
	return (await response.json()) as Usage;
}

// Starts a subscription paid by the Mercado Pago account of payerEmail.
// Answers where to send the user to authorize it.
export async function subscribe(plan: Plan, payerEmail: string) {
	const response = await ensure(
		await api.post('/v1/billing/subscribe', { plan, payer_email: payerEmail })
	);
	return ((await response.json()) as { init_point: string }).init_point;
}

// Asks the payment provider how the caller's subscription stands now.
export async function sync() {
	const response = await ensure(await api.post('/v1/billing/sync', undefined));
	return (await response.json()) as Subscription;
}

// An upgrade applies at once, a downgrade on the next charge.
export async function change(plan: Plan) {
	const response = await ensure(await api.put('/v1/billing/subscription', { plan }));
	return (await response.json()) as Subscription;
}

// The plan lasts until the subscription's `expires`.
export async function cancel() {
	await ensure(await api.del('/v1/billing/subscription'));
}
