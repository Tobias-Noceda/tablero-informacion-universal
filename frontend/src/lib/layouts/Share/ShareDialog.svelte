<script lang="ts">
	import Button from '$components/Button/Button.svelte';
	import Input from '$components/Input/Input.svelte';
	import Modal from '$components/Modal/Modal.svelte';
	import * as boardApi from '$services/board';
	import { session } from '$modules/session.svelte';
	import { m } from '$lib/paraglide/messages';
	import type { BoardMember, BoardMemberSummary, BoardRole, UUID } from '$types/api';
	import Dropdown from '$components/Dropdown/Dropdown.svelte';
	import { boardPlan } from '$stores/plan.svelte';
	import { canInvite } from '$lib/billing/can';
	import PlanHint from '$components/PlanHint/PlanHint.svelte';

	let {
		board,
		role,
		onclose,
		onleave
	}: { board: UUID; role: BoardRole; onclose: () => void; onleave: () => void } = $props();

	const isOwner = $derived(role === 'owner');
	const assignable: BoardMember['role'][] = ['editor', 'viewer'];
	// Once every seat of the plan is taken (always on a plan that does not
	// share) there is no form to add someone; the list stays.
	const canAdd = $derived(canInvite(boardPlan.limits, boardPlan.seatsUsed));
	// A plan that shares read-only (Free) never offers the editor role; an
	// editor from a former plan keeps it and can only be made a viewer.
	const readOnly = $derived(boardPlan.limits.viewers_only);
	const rolesFor = (current: BoardMember['role']) =>
		readOnly
			? assignable.filter((option) => option === 'viewer' || option === current)
			: assignable;

	let members = $state<BoardMemberSummary[]>([]);
	// Someone who reaches the board through its organization is not listed
	// and has nothing to leave.
	const listed = $derived(members.some((member) => member.user.id === session.userId));
	let loading = $state(true);
	let error = $state('');

	let email = $state('');
	let newRole = $state<BoardMember['role']>('viewer');

	const errors: Record<string, () => string> = {
		user_not_found: m['members.error_user_not_found'],
		invalid_role: m['members.error_invalid_role'],
		owner_role: m['members.error_owner_role'],
		rate_limited: m['members.error_rate_limited'],
		plan_limit: m['billing.limit.title']
	};

	function describe(err: unknown) {
		const message = err instanceof boardApi.ShareError ? errors[err.code] : undefined;
		return (message ?? m['members.error_unknown'])();
	}

	async function attempt(action: () => Promise<unknown>) {
		error = '';
		try {
			await action();
			return true;
		} catch (err) {
			error = describe(err);
			return false;
		}
	}

	async function refresh() {
		loading = true;
		await attempt(async () => (members = await boardApi.members(board)));
		loading = false;
	}

	async function add(event: SubmitEvent) {
		event.preventDefault();
		if (
			await attempt(() => boardApi.setMember(board, email.trim(), readOnly ? 'viewer' : newRole))
		) {
			email = '';
			await refresh();
		}
		// A seat was taken, or the backend said there is none left.
		await boardPlan.reload();
	}

	async function change(member: BoardMemberSummary, to: BoardMember['role']) {
		await attempt(() => boardApi.setMember(board, member.user.email, to));
		await refresh();
	}

	async function remove(member: BoardMemberSummary) {
		if (await attempt(() => boardApi.removeMember(board, member.user.id))) {
			await refresh();
			await boardPlan.reload();
		}
	}

	async function leave() {
		if (await attempt(() => boardApi.removeMember(board, session.userId))) onleave();
	}

	$effect(() => {
		void board;
		refresh();
	});
</script>

<Modal {onclose} onaccept={onclose} acceptText={m['secrets.close']()}>
	<h2 class="text-lg font-semibold">{m['members.title']()}</h2>

	{#if role === 'viewer'}
		<p class="text-xs opacity-70">{m['members.viewer_hint']()}</p>
	{/if}

	{#if error}
		<p role="alert" class="text-sm text-destructive">{error}</p>
	{/if}

	{#if loading}
		<p class="text-sm opacity-70">{m['members.loading']()}</p>
	{:else}
		<ul class="flex flex-col gap-2">
			{#each members as member (member.user.id)}
				<li
					class="flex items-center justify-between gap-2 rounded-md border border-main-border px-3 py-2"
				>
					<div class="flex min-w-0 flex-col">
						<span class="truncate text-sm">
							{member.user.name || member.user.email || member.user.id}
							{#if member.user.id === session.userId}
								<span class="opacity-60">({m['members.you']()})</span>
							{/if}
						</span>
						<span class="truncate text-xs opacity-60">{member.user.email}</span>
					</div>
					<div class="flex items-center gap-2">
						{#if isOwner && member.role !== 'owner' && rolesFor(member.role).length > 1}
							<select
								class="rounded-md border border-main-border bg-background px-2 py-1 text-sm"
								aria-label={m['members.title']()}
								value={member.role}
								onchange={(e) => change(member, e.currentTarget.value as BoardMember['role'])}
							>
								{#each rolesFor(member.role) as option (option)}
									<option value={option}>{m[`members.role_${option}`]()}</option>
								{/each}
							</select>
							<Button variant="destructive" onclick={() => remove(member)}
								>{m['members.remove']()}</Button
							>
						{:else if isOwner && member.role !== 'owner'}
							<span class="text-sm opacity-70">{m[`members.role_${member.role}`]()}</span>
							<Button variant="destructive" onclick={() => remove(member)}
								>{m['members.remove']()}</Button
							>
						{:else}
							<span class="text-sm opacity-70">{m[`members.role_${member.role}`]()}</span>
						{/if}
					</div>
				</li>
			{/each}
		</ul>
	{/if}

	{#if isOwner && !canAdd}
		<PlanHint
			class="border-t border-main-border pt-3"
			text={m['billing.hint.seats']({
				count: boardPlan.seatsUsed ?? boardPlan.limits.seats,
				max: boardPlan.limits.seats
			})}
		/>
	{:else if isOwner}
		<form class="flex flex-col gap-2 border-t border-main-border pt-3" onsubmit={add}>
			<Input
				label={m['members.email']()}
				type="email"
				autocomplete="off"
				bind:value={email}
				required
			/>
			<div class="flex w-fit! gap-2">
				{#if !readOnly}
					<Dropdown
						floating
						variant="plain"
						options={assignable.map((role) => ({
							label: m[`members.role_${role}`](),
							value: role
						}))}
						class="w-fit!"
						activatorClass={() => 'w-fit!'}
						bind:value={newRole}
					/>
				{/if}
				<Button variant="primary" type="submit" disabled={email.trim() === ''}
					>{m['members.add']()}</Button
				>
			</div>
			{#if readOnly}
				<PlanHint text={m['billing.hint.read_only']()} />
			{/if}
		</form>
	{:else if listed}
		<Button variant="destructive" onclick={leave}>{m['members.leave']()}</Button>
	{/if}
</Modal>
