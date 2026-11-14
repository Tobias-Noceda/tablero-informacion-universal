<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import Button from '$components/Button/Button.svelte';
	import Icon from '$components/Icon/Icon.svelte';
	import Input from '$components/Input/Input.svelte';
	import { money, planName } from '$lib/billing/format';
	import { m } from '$lib/paraglide/messages';
	import { session } from '$modules/session.svelte';
	import * as billing from '$services/billing';
	import { plan as own } from '$stores/plan.svelte';
	import type { Limits, Plan, PlanCatalogue, PlanInfo, Subscription } from '$types/api';

	let catalogue = $state<PlanCatalogue | null>(null);
	let loadFailed = $state(false);
	let subscription = $state<Subscription | null>(null);

	// The plan the payer's email is being asked for.
	let choosing = $state<Plan | null>(null);
	let payerEmail = $state('');
	let busy = $state(false);
	let error = $state('');
	let notice = $state('');

	const signedIn = $derived(session.status === 'authenticated');

	// Back to wherever the visitor came from; a page opened directly (no
	// history in this tab) goes to the app or, signed out, to the about page.
	function goBack() {
		if (history.length > 1) history.back();
		else void goto(resolve(signedIn ? '/' : '/about'));
	}
	// A live subscription changes plan in place instead of starting another.
	const subscribed = $derived(subscription?.status === 'authorized');
	// Paused counts as subscribed too, but cannot be changed or restarted from
	// here: it is resumed at Mercado Pago.
	const paused = $derived(subscription?.status === 'paused');

	async function load() {
		try {
			catalogue = await billing.plans();
		} catch (err) {
			console.error(err);
			loadFailed = true;
		}
	}

	async function loadSubscription() {
		try {
			subscription = await billing.subscription();
		} catch {
			subscription = null;
		}
	}

	$effect(() => {
		if (!signedIn) {
			subscription = null;
			return;
		}
		payerEmail ||= session.user?.email ?? '';
		void loadSubscription();
	});

	void load();

	function features(limits: Limits) {
		const list: string[] = [];
		list.push(
			limits.orgs > 0
				? m['pricing.feature.orgs']({ count: limits.orgs })
				: m['pricing.feature.orgs_none']()
		);
		list.push(m['pricing.feature.boards']({ count: limits.boards_per_space }));
		list.push(
			limits.seats <= 1
				? m['pricing.feature.seats_none']()
				: limits.viewers_only
					? m['pricing.feature.seats_read_only']({ count: limits.seats - 1 })
					: m['pricing.feature.seats']({ count: limits.seats })
		);
		list.push(m['pricing.feature.nodes']({ count: limits.nodes_per_board }));
		list.push(m['pricing.feature.templates_user']({ count: limits.user_templates }));
		if (limits.board_templates > 0 || limits.org_templates > 0) {
			list.push(
				m['pricing.feature.templates_shared']({
					board: limits.board_templates,
					org: limits.org_templates
				})
			);
		}
		list.push(m['pricing.feature.secrets']({ count: limits.secrets }));
		if (limits.groups > 0) list.push(m['pricing.feature.groups']({ count: limits.groups }));
		list.push(m['pricing.feature.min_rate']({ seconds: limits.min_rate }));
		if (limits.force_refresh) list.push(m['pricing.feature.force_refresh']());
		if (limits.api_access) list.push(m['pricing.feature.api_access']());
		return list;
	}

	function describe(err: unknown) {
		const code = err instanceof billing.BillingError ? err.code : 'unknown';
		const key = `pricing.error.${code}` as keyof typeof m;
		const message = (m as Record<string, unknown>)[key];
		return typeof message === 'function'
			? (message as () => string)()
			: m['pricing.error.unknown']();
	}

	function choose(info: PlanInfo) {
		error = '';
		notice = '';
		if (subscribed) void change(info.plan);
		else choosing = info.plan;
	}

	async function change(target: Plan) {
		busy = true;
		try {
			const updated = await billing.change(target);
			subscription = updated;
			notice =
				updated.pending_plan === target
					? m['pricing.switched_later']({ plan: planName(target) })
					: m['pricing.switched_now']({ plan: planName(target) });
			await own.reloadProfile().catch(() => {});
		} catch (err) {
			error = describe(err);
		} finally {
			busy = false;
		}
	}

	async function start(event: SubmitEvent) {
		event.preventDefault();
		if (!choosing) return;
		error = '';
		busy = true;
		try {
			window.location.assign(await billing.subscribe(choosing, payerEmail.trim()));
		} catch (err) {
			error = describe(err);
			busy = false;
		}
	}
</script>

<svelte:head>
	<title>{m['pricing.title']()}</title>
</svelte:head>

<main class="mx-auto flex w-full max-w-5xl flex-col gap-8 p-4 md:p-6">
	<header
		class="relative flex flex-col gap-3 rounded-2xl bg-linear-to-br from-main/15 via-sidebar/20 to-transparent p-6 text-center md:p-10"
	>
		<button
			type="button"
			class="absolute top-3 left-3 flex h-9 w-9 cursor-pointer items-center justify-center rounded-full transition-colors hover:bg-main/15 md:top-4 md:left-4"
			aria-label={m['pricing.back']()}
			title={m['pricing.back']()}
			onclick={goBack}
		>
			<Icon name="left-arrow" class="h-5 w-5" />
		</button>
		<h1 class="text-4xl font-bold tracking-tight">{m['pricing.title']()}</h1>
		<p class="text-lg font-medium">{m['pricing.hero']()}</p>
		<p class="mx-auto max-w-2xl opacity-80">{m['pricing.subtitle']()}</p>
		<a href={resolve(signedIn ? '/' : '/about')} class="text-sm underline hover:text-main">
			{signedIn ? m['pricing.to_app']() : m['pricing.back']()}
		</a>
	</header>

	{#if notice}
		<p role="status" class="text-sm text-success">{notice}</p>
	{/if}
	{#if paused}
		<p class="text-sm opacity-70">{m['pricing.paused']()}</p>
	{/if}

	{#if loadFailed}
		<p role="alert" class="text-destructive">{m['pricing.load_failed']()}</p>
	{:else if !catalogue}
		<p class="text-sm opacity-70">{m['pricing.loading']()}</p>
	{:else}
		<div class="grid gap-4 md:grid-cols-3">
			{#each catalogue.plans as info (info.plan)}
				{@const current = signedIn && own.name === info.plan}
				{@const featured = info.plan === 'pro'}
				<section
					class="relative flex flex-col overflow-hidden rounded-2xl border bg-background shadow-lg transition hover:-translate-y-1 hover:shadow-xl {featured
						? 'z-10 border-main ring-2 ring-main md:scale-105'
						: current
							? 'border-main'
							: 'border-border'}"
				>
					<div
						class="flex items-center justify-between gap-2 px-5 py-3 {info.plan === 'pro'
							? 'bg-main text-main-text'
							: info.plan === 'max'
								? 'bg-linear-to-r from-main to-sidebar-border text-main-text'
								: 'bg-sidebar/30'}"
					>
						<h2 class="text-xl font-semibold">{planName(info.plan)}</h2>
						<div class="flex flex-wrap justify-end gap-1">
							{#if featured}
								<span
									class="inline-flex items-center gap-1 rounded-full bg-background px-2 py-0.5 text-xs font-semibold text-foreground"
								>
									<Icon name="sparkles" class="h-3 w-3" />
									{m['pricing.popular']()}
								</span>
							{/if}
							{#if current}
								<span
									class="rounded-full bg-main px-2 py-0.5 text-xs font-semibold text-main-text ring-1 ring-main-text/40"
								>
									{m['pricing.your_plan']()}
								</span>
							{/if}
						</div>
					</div>
					<div class="flex flex-1 flex-col gap-4 p-5">
						<div>
							<p class="flex items-baseline gap-1">
								<span class="text-4xl font-bold">
									{info.price_usd > 0 ? money(info.price_usd, 'USD') : m['pricing.price_free']()}
								</span>
								{#if info.price_usd > 0}
									<span class="text-sm opacity-70">{m['pricing.month']()}</span>
								{/if}
							</p>
							{#if info.price_usd > 0 && info.price_ars > 0}
								<p class="text-xs opacity-60">
									{m['pricing.price_ars']({ amount: money(info.price_ars, catalogue.currency) })}
								</p>
							{/if}
						</div>
						<ul class="flex flex-col gap-2 text-sm">
							{#each features(info.limits) as line (line)}
								<li class="flex items-start gap-2">
									<Icon name="check" class="mt-0.5 h-4 w-4 shrink-0 text-success" />
									<span>{line}</span>
								</li>
							{/each}
						</ul>

						<div class="mt-auto flex flex-col gap-2 pt-2">
							{#if current}
								{#if subscription?.pending_plan}
									<p class="text-xs opacity-70">
										{m['pricing.switched_later']({ plan: planName(subscription.pending_plan) })}
									</p>
								{/if}
							{:else if info.plan !== 'free' && catalogue.checkout}
								{#if !signedIn}
									<div class="flex flex-col gap-2 text-sm">
										<a
											href="{resolve('/register')}?next=/pricing"
											class="rounded-md bg-main px-4 py-2 text-center font-semibold text-main-text hover:bg-main-hover"
										>
											{m['pricing.register']()}
										</a>
										<a href="{resolve('/login')}?next=/pricing" class="text-center underline">
											{m['pricing.sign_in']()}
										</a>
									</div>
								{:else if choosing === info.plan}
									<form class="flex flex-col gap-2" onsubmit={start}>
										<Input
											type="email"
											label={m['pricing.payer_email']()}
											bind:value={payerEmail}
											required
										/>
										<p class="text-xs opacity-70">{m['pricing.payer_email_hint']()}</p>
										{#if error}
											<p role="alert" class="text-sm text-destructive">{error}</p>
										{/if}
										<div class="flex gap-2">
											<Button type="submit" variant="primary" class="flex-1" disabled={busy}>
												{m['pricing.continue']()}
											</Button>
											<Button type="button" variant="gray" onclick={() => (choosing = null)}>
												{m['pricing.back']()}
											</Button>
										</div>
									</form>
								{:else if !paused && (own.name === 'free' || !subscribed || info.plan !== own.name)}
									<Button
										type="button"
										variant={featured ? 'primary' : 'secondary'}
										class="w-full"
										disabled={busy}
										onclick={() => choose(info)}
									>
										{subscribed
											? m['pricing.switch']({ plan: planName(info.plan) })
											: m['pricing.choose']({ plan: planName(info.plan) })}
									</Button>
								{/if}
							{/if}
						</div>
					</div>
				</section>
			{/each}
		</div>
		{#if error && !choosing}
			<p role="alert" class="text-sm text-destructive">{error}</p>
		{/if}
	{/if}
</main>
