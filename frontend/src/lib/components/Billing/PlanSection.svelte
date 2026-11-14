<script lang="ts">
	import { resolve } from '$app/paths';
	import Modal from '$components/Modal/Modal.svelte';
	import { day, money, planName } from '$lib/billing/format';
	import { m } from '$lib/paraglide/messages';
	import { session } from '$modules/session.svelte';
	import * as billing from '$services/billing';
	import { plan } from '$stores/plan.svelte';
	import type { Subscription } from '$types/api';

	// The profile's "Plan" section: what the user is on, when it is charged
	// next, and the way out of a subscription.
	let subscription = $state<Subscription | null>(null);
	let confirming = $state(false);
	let busy = $state(false);
	let notice = $state<'canceled' | 'failed' | null>(null);

	// Until when the plan lasts once it is not charged again.
	const expires = $derived(subscription?.expires ?? session.user?.plan_expires);
	const paused = $derived(subscription?.status === 'paused');
	// A paused subscription is still one: it is resumed at Mercado Pago.
	const subscribed = $derived(subscription?.status === 'authorized' || paused);
	const charging = $derived(subscription?.status === 'authorized');
	const ending = $derived(
		plan.name !== 'free' &&
			(subscription?.status === 'canceled' || subscription?.status === 'paused')
	);

	async function load() {
		try {
			subscription = await billing.subscription();
		} catch {
			// Shown as "no subscription": the plan itself comes with the profile.
			subscription = null;
		}
	}

	async function cancel() {
		busy = true;
		notice = null;
		try {
			await billing.cancel();
			notice = 'canceled';
		} catch (err) {
			console.error(err);
			notice = 'failed';
		} finally {
			busy = false;
			confirming = false;
		}
		await Promise.all([load(), plan.reloadProfile().catch(() => {})]);
	}

	load();
</script>

<section
	class="flex flex-col gap-3 rounded-2xl border border-l-4 p-4 shadow-sm {plan.name === 'max'
		? 'border-main bg-linear-to-br from-main/15 to-sidebar-border/20'
		: plan.name === 'pro'
			? 'border-main bg-main/10'
			: 'border-border border-l-sidebar bg-sidebar/10'}"
>
	<h2 class="text-sm font-medium">{m['billing.profile.title']()}</h2>
	<div class="flex flex-wrap items-center gap-3">
		<span
			class="rounded-full px-4 py-1 text-lg font-bold {plan.name === 'max'
				? 'bg-linear-to-r from-main to-sidebar-border text-main-text'
				: plan.name === 'pro'
					? 'bg-main text-main-text'
					: 'bg-sidebar/40'}"
		>
			{planName(plan.name)}
		</span>
		<p>{m['billing.profile.current']({ plan: planName(plan.name) })}</p>
	</div>

	{#if charging && subscription?.next_payment}
		<p class="text-sm opacity-70">
			{m['billing.profile.next_charge']({
				date: day(subscription.next_payment),
				amount: money(subscription.amount_ars ?? 0, 'ARS')
			})}
		</p>
	{/if}
	{#if charging && subscription?.pending_plan}
		<p class="text-sm opacity-70">
			{m['billing.profile.pending_plan']({ plan: planName(subscription.pending_plan) })}
		</p>
	{/if}
	{#if paused}
		<p class="text-sm opacity-70">{m['billing.profile.paused']()}</p>
	{/if}
	{#if subscription?.status === 'pending'}
		<p class="text-sm opacity-70">{m['billing.profile.pending']()}</p>
	{/if}
	{#if ending && expires}
		<p class="text-sm opacity-70">
			{m['billing.profile.ends']({ plan: planName(plan.name), date: day(expires) })}
		</p>
	{/if}

	{#if notice === 'canceled'}
		<p class="text-sm text-success">{m['billing.profile.canceled']()}</p>
	{:else if notice === 'failed'}
		<p role="alert" class="text-sm text-destructive">{m['billing.profile.cancel_failed']()}</p>
	{/if}

	{#if plan.name === 'free'}
		<p class="text-sm opacity-80">{m['billing.profile.unlocks']()}</p>
	{/if}

	<div class="flex flex-wrap items-center gap-3">
		<a
			href={resolve('/pricing')}
			class="rounded-md px-4 py-2 text-center font-semibold {plan.name === 'max'
				? 'border border-main bg-background text-main hover:bg-main hover:text-main-text'
				: 'border border-main-border bg-main text-main-text hover:bg-main-hover'}"
		>
			{plan.name === 'free'
				? m['billing.profile.upgrade']()
				: plan.name === 'max'
					? m['billing.profile.manage']()
					: charging
						? m['billing.profile.change']()
						: m['billing.profile.see_plans']()}
		</a>
		{#if subscribed}
			<button
				type="button"
				class="cursor-pointer rounded-md border border-destructive bg-transparent px-4 py-2 text-sm font-semibold text-destructive transition-colors hover:bg-destructive hover:text-white"
				onclick={() => (confirming = true)}
			>
				{m['billing.profile.cancel']()}
			</button>
		{/if}
	</div>
</section>

{#if confirming}
	<Modal
		onclose={() => (confirming = false)}
		onaccept={cancel}
		acceptText={m['billing.profile.cancel']()}
		acceptDisabled={busy}
	>
		<h2 class="text-lg font-semibold">{m['billing.profile.cancel_title']()}</h2>
		<p class="text-sm">
			{expires
				? m['billing.profile.cancel_note_until']({ plan: planName(plan.name), date: day(expires) })
				: m['billing.profile.cancel_note']()}
		</p>
	</Modal>
{/if}
