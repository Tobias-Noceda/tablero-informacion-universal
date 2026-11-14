import { describe, expect, it } from 'vitest';
import type { Limits } from '$types/api';
import {
	FREE_LIMITS,
	canAddNode,
	canAddOwnSecret,
	canAddSecret,
	canAddTemplate,
	canCreateBoard,
	canCreateGroup,
	canCreateOrg,
	canInvite,
	canRefresh,
	hasApi,
	limitsOf,
	offersTemplates,
	orgLimits,
	paysFor
} from './can';

const PRO: Limits = {
	orgs: 3,
	boards_per_space: 10,
	seats: 5,
	viewers_only: false,
	nodes_per_board: 100,
	user_templates: 10,
	board_templates: 5,
	org_templates: 3,
	secrets: 25,
	groups: 5,
	min_rate: 150,
	force_refresh: false,
	api_access: true,
	price_usd: 9
};

const MAX: Limits = { ...PRO, orgs: 10, seats: 25, force_refresh: true, price_usd: 29 };

describe('limitsOf', () => {
	it('falls back to Free when nothing carries limits', () => {
		expect(limitsOf(null)).toBe(FREE_LIMITS);
		expect(limitsOf({})).toBe(FREE_LIMITS);
		expect(limitsOf({ limits: PRO })).toBe(PRO);
	});
});

describe('numeric limits', () => {
	it('allow one more below the limit and none at it', () => {
		expect(canCreateBoard(FREE_LIMITS, 4)).toBe(true);
		expect(canCreateBoard(FREE_LIMITS, 5)).toBe(false);
		expect(canCreateBoard(PRO, 9)).toBe(true);
		expect(canCreateBoard(PRO, 10)).toBe(false);

		expect(canAddNode(FREE_LIMITS, 24)).toBe(true);
		expect(canAddNode(FREE_LIMITS, 25)).toBe(false);
		expect(canAddNode(PRO, 99)).toBe(true);
	});

	it('still refuse once an account is over its limit', () => {
		expect(canCreateBoard(FREE_LIMITS, 12)).toBe(false);
	});

	it('leave an unknown count to the backend unless the plan has none', () => {
		expect(canCreateBoard(FREE_LIMITS)).toBe(true);
		expect(canCreateOrg(PRO)).toBe(true);
		expect(canCreateOrg(FREE_LIMITS)).toBe(false);
	});
});

describe('seats', () => {
	it('lets Free share with four more people, read-only', () => {
		expect(FREE_LIMITS.viewers_only).toBe(true);
		expect(canInvite(FREE_LIMITS, 4)).toBe(true);
		expect(canInvite(FREE_LIMITS, 5)).toBe(false);
	});

	it('counts the owner', () => {
		expect(canInvite(PRO, 4)).toBe(true);
		expect(canInvite(PRO, 5)).toBe(false);
		expect(canInvite(MAX, 24)).toBe(true);
	});
});

describe('organizations and groups', () => {
	it('are not part of Free', () => {
		expect(canCreateOrg(FREE_LIMITS, 0)).toBe(false);
		expect(canCreateGroup(FREE_LIMITS, 0)).toBe(false);
	});

	it('stop at the plan limit', () => {
		expect(canCreateOrg(PRO, 2)).toBe(true);
		expect(canCreateOrg(PRO, 3)).toBe(false);
		expect(canCreateGroup(PRO, 5)).toBe(false);
	});
});

describe('post-it templates', () => {
	it('offers only the user scope on Free', () => {
		expect(offersTemplates('user', FREE_LIMITS)).toBe(true);
		expect(offersTemplates('board', FREE_LIMITS)).toBe(false);
		expect(offersTemplates('org', FREE_LIMITS)).toBe(false);
		expect(offersTemplates('board', PRO)).toBe(true);
	});

	it('uses the limit of each scope', () => {
		expect(canAddTemplate('user', FREE_LIMITS, 4)).toBe(true);
		expect(canAddTemplate('user', FREE_LIMITS, 5)).toBe(false);
		expect(canAddTemplate('board', FREE_LIMITS, 0)).toBe(false);
		expect(canAddTemplate('board', PRO, 4)).toBe(true);
		expect(canAddTemplate('org', PRO, 3)).toBe(false);
	});
});

describe('secrets', () => {
	it('follow what the board payer has left', () => {
		expect(canAddSecret(1)).toBe(true);
		expect(canAddSecret(0)).toBe(false);
		expect(canAddSecret(undefined)).toBe(true);
	});

	it("follow the caller's own usage", () => {
		expect(canAddOwnSecret(FREE_LIMITS, 4)).toBe(true);
		expect(canAddOwnSecret(FREE_LIMITS, 5)).toBe(false);
		expect(canAddOwnSecret(FREE_LIMITS)).toBe(true);
	});
});

describe('features', () => {
	it('refresh only on Max', () => {
		expect(canRefresh(FREE_LIMITS)).toBe(false);
		expect(canRefresh(PRO)).toBe(false);
		expect(canRefresh(MAX)).toBe(true);
	});

	it('API access on paid plans', () => {
		expect(hasApi(FREE_LIMITS)).toBe(false);
		expect(hasApi(PRO)).toBe(true);
	});
});

describe('paysFor', () => {
	const members = [
		{ user: 'ana', role: 'admin' as const },
		{ user: 'beto', role: 'member' as const }
	];

	it('is the owner when the organization names one', () => {
		expect(paysFor({ owner: 'ana', members }, 'ana')).toBe(true);
		expect(paysFor({ owner: 'ana', members }, 'beto')).toBe(false);
	});

	it('is the only admin of an organization without owner', () => {
		expect(paysFor({ members }, 'ana')).toBe(true);
		expect(paysFor({ members }, 'beto')).toBe(false);
	});

	it('reads member summaries too', () => {
		const detailed = [{ user: { id: 'ana', name: 'Ana', email: 'a@x' }, role: 'admin' as const }];
		expect(paysFor({ members: detailed }, 'ana')).toBe(true);
	});

	it('cannot tell between several admins', () => {
		const two = [...members, { user: 'caro', role: 'admin' as const }];
		expect(paysFor({ members: two }, 'ana')).toBeUndefined();
	});

	it('prefers the limits the backend reports for the organization', () => {
		const reported = { ...PRO, seats: 7 };
		expect(orgLimits({ owner: 'ana', limits: reported, members }, 'beto', FREE_LIMITS)).toBe(
			reported
		);
	});

	it('gives limits only to the payer', () => {
		expect(orgLimits({ owner: 'ana', members }, 'ana', PRO)).toBe(PRO);
		expect(orgLimits({ owner: 'ana', members }, 'beto', FREE_LIMITS)).toBeUndefined();
	});
});
