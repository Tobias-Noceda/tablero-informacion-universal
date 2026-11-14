import type { Limits, OrgRole, PostitTemplateScope, UserSummary, UUID } from '$types/api';

// The Free plan, for a backend that does not send limits yet. Mirrors
// models.LimitsFor in the backend.
export const FREE_LIMITS: Limits = {
	orgs: 0,
	boards_per_space: 5,
	seats: 5,
	viewers_only: true,
	nodes_per_board: 25,
	user_templates: 5,
	board_templates: 0,
	org_templates: 0,
	secrets: 5,
	groups: 0,
	min_rate: 600,
	force_refresh: false,
	api_access: false,
	price_usd: 0
};

// The limits of a user or a board, Free when it carries none.
export function limitsOf(holder: { limits?: Limits | null } | null | undefined): Limits {
	return holder?.limits ?? FREE_LIMITS;
}

// Whether one more fits. An unknown count only refuses what the plan never
// has: the backend answers the rest.
function fits(max: number, count: number | undefined) {
	return max > 0 && (count === undefined || count < max);
}

// `count`: boards of the space (the user's own without an organization, or
// the organization's).
export function canCreateBoard(limits: Limits, count?: number) {
	return fits(limits.boards_per_space, count);
}

export function canAddNode(limits: Limits, count?: number) {
	return fits(limits.nodes_per_board, count);
}

// `seatsUsed`: people on the board or the organization, its owner included.
export function canInvite(limits: Limits, seatsUsed?: number) {
	return fits(limits.seats, seatsUsed);
}

export function canCreateOrg(limits: Limits, owned?: number) {
	return fits(limits.orgs, owned);
}

export function canCreateGroup(limits: Limits, owned?: number) {
	return fits(limits.groups, owned);
}

type TemplateKind = PostitTemplateScope['kind'];

export function templateLimit(kind: TemplateKind, limits: Limits) {
	return limits[`${kind}_templates`];
}

// Whether a scope has post-it templates at all on this plan.
export function offersTemplates(kind: TemplateKind, limits: Limits) {
	return templateLimit(kind, limits) > 0;
}

export function canAddTemplate(kind: TemplateKind, limits: Limits, count?: number) {
	return fits(templateLimit(kind, limits), count);
}

// `left`: what the board's payer may still store (Board.secrets_left).
export function canAddSecret(left: number | undefined) {
	return left === undefined || left > 0;
}

// The caller's own credentials, against their own plan.
export function canAddOwnSecret(limits: Limits, used?: number) {
	return fits(limits.secrets, used);
}

export function canRefresh(limits: Limits) {
	return limits.force_refresh;
}

export function hasApi(limits: Limits) {
	return limits.api_access;
}

type OrgLike = {
	owner?: UUID;
	// The payer's, as the backend reports them.
	limits?: Limits;
	members: { user: UUID | UserSummary; role: OrgRole }[];
};

// Whether `user` is the one whose plan an organization runs on: its owner,
// or, on one from before owners existed, its only admin. Undefined when the
// payer cannot be told from here.
export function paysFor(org: OrgLike, user: UUID): boolean | undefined {
	if (org.owner) return org.owner === user;
	const admins = org.members.filter((member) => member.role === 'admin');
	if (admins.length !== 1) return undefined;
	const admin = admins[0].user;
	return (typeof admin === 'string' ? admin : admin.id) === user;
}

// The limits an organization runs on: the ones the backend reports (its
// payer's, whoever asks), else the caller's when they are known to be the
// payer. Undefined means "ask the backend": nothing is hidden on a guess.
export function orgLimits(org: OrgLike, user: UUID, own: Limits): Limits | undefined {
	if (org.limits) return org.limits;
	return paysFor(org, user) ? own : undefined;
}
