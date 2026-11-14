import { FREE_LIMITS, canCreateBoard, limitsOf, orgLimits } from '$lib/billing/can';
import { m } from '$lib/paraglide/messages';
import * as api from '$modules/api.svelte';
import { session } from '$modules/session.svelte';
import * as billing from '$services/billing';
import * as boardApi from '$services/board';
import * as users from '$services/users';
import type { Board, Limits, Profile, Usage, UUID } from '$types/api';
import { refreshBoards } from './boards';
import { orgs } from './org.svelte';
import { toast } from './toast.svelte';

// What the signed-in user's own plan allows and how much of it they use.
// The limits travel with the profile; the usage is asked for on demand and
// stays unknown (undefined) if the backend cannot tell.
class OwnPlan {
	usage = $state<Usage | undefined>();
	#generation = 0;

	get limits(): Limits {
		return limitsOf(session.user);
	}

	get name() {
		return session.user?.plan ?? 'free';
	}

	// The profile again, after something that may have changed the plan.
	async reloadProfile() {
		const id = session.user?.id;
		if (!id) return;
		const profile = (await users.get(id)) as Profile;
		if (session.user?.id === id) session.user = profile;
	}

	async refreshUsage() {
		const generation = this.#generation;
		try {
			const usage = await billing.usage();
			if (generation === this.#generation) this.usage = usage;
		} catch {
			if (generation === this.#generation) this.usage = undefined;
		}
	}

	reset() {
		this.#generation++;
		this.usage = undefined;
	}
}

// What the open board allows: the plan of whoever pays for it, which is not
// necessarily the caller's. Filled by the board's load.
class BoardPlan {
	limits = $state<Limits>(FREE_LIMITS);
	// People on the board, its owner included.
	seatsUsed = $state<number | undefined>();
	// Credentials its payer may still store.
	secretsLeft = $state<number | undefined>();
	// Cards on the canvas right now (kept by Flow).
	nodes = $state(0);
	#board: UUID | undefined;

	adopt(board: Board) {
		this.#board = board.id;
		this.limits = limitsOf(board);
		this.seatsUsed = board.seats_used;
		this.secretsLeft = board.secrets_left;
		this.nodes = board.postits.length;
	}

	// After adding or removing people or credentials, or a refused change.
	async reload() {
		const id = this.#board;
		if (!id) return;
		try {
			const board = await boardApi.get(id);
			if (this.#board !== id) return;
			this.limits = limitsOf(board);
			this.seatsUsed = board.seats_used;
			this.secretsLeft = board.secrets_left;
		} catch {
			// The values on screen stay; the backend refuses what no longer fits.
		}
	}

	reset() {
		this.#board = undefined;
		this.limits = FREE_LIMITS;
		this.seatsUsed = undefined;
		this.secretsLeft = undefined;
		this.nodes = 0;
	}
}

export const plan = new OwnPlan();
export const boardPlan = new BoardPlan();

// Whether one more board fits in a space (an organization, or Personal when
// there is none), out of the boards the user sees. Personal counts the ones
// they own; an organization's run on its payer's plan, which it reports
// (`limits`, `boards_used`). Without that the creator stays and the backend
// answers.
export function boardSpace(orgId: UUID | undefined, boards: Board[]) {
	const me = session.user?.id;
	if (!orgId) {
		const count = boards.filter((board) => !board.org && board.owner === me).length;
		const max = plan.limits.boards_per_space;
		return { allowed: canCreateBoard(plan.limits, count), count, max };
	}

	const org = orgs.byId(orgId);
	// The organization's own count when it reports one: it covers boards the
	// caller does not see.
	const count = org?.boards_used ?? boards.filter((board) => board.org === orgId).length;
	const limits = org && me ? orgLimits(org, me, plan.limits) : undefined;
	return {
		allowed: limits ? canCreateBoard(limits, count) : true,
		count,
		max: limits?.boards_per_space
	};
}

session.onUserChange(() => {
	plan.reset();
	boardPlan.reset();
});

const refusals: Record<string, (inputs: { max: number }) => string> = {
	orgs: m['billing.limit.orgs'],
	boards: m['billing.limit.boards'],
	seats: m['billing.limit.seats'],
	editors: m['billing.limit.editors'],
	nodes: m['billing.limit.nodes'],
	user_templates: m['billing.limit.templates'],
	board_templates: m['billing.limit.templates'],
	org_templates: m['billing.limit.templates'],
	secrets: m['billing.limit.secrets'],
	groups: m['billing.limit.groups'],
	force_refresh: m['billing.limit.force_refresh'],
	api_access: m['billing.limit.api_access']
};

// The backend refused something the screen still offered (another tab, a
// plan that just ended): say so and read again what the plan allows, so the
// entry point goes away.
api.onPlanLimit(({ limit, max }) => {
	toast.notify({
		title: m['billing.limit.title'](),
		description: (refusals[limit] ?? m['billing.limit.generic'])({ max })
	});
	void plan.reloadProfile().catch(() => {});
	void plan.refreshUsage();
	void boardPlan.reload();
	void refreshBoards().catch(() => {});
	void orgs.refresh().catch(() => {});
});
