import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { Board, Org } from '$types/api';

const { list } = vi.hoisted(() => {
	const acme: Org = { id: 'acme', name: 'Acme', members: [], created_at: '', role: 'member' };
	return { list: vi.fn(async () => [acme]) };
});
vi.mock('$services/orgs', () => ({ list }));

const { orgs } = await import('./org.svelte');

function board(org?: string): Board {
	return { id: 'b', name: 'B', owner: 'o', members: [], org, postits: [], strands: [], envs: [] };
}

describe('organizations', () => {
	beforeEach(async () => {
		await orgs.refresh();
	});

	it('shows personal boards under Personal', () => {
		expect(orgs.shows(board())).toBe(true);
		expect(orgs.shows(board('acme'))).toBe(false);
	});

	it('shows only that organization’s boards under it', () => {
		expect(orgs.byId('acme')?.name).toBe('Acme');
		expect(orgs.shows(board('acme'), 'acme')).toBe(true);
		expect(orgs.shows(board(), 'acme')).toBe(false);
	});

	it('files a board shared from an organization the user is not in as personal', () => {
		expect(orgs.orgOf(board('elsewhere'))).toBeUndefined();
		expect(orgs.shows(board('elsewhere'))).toBe(true);
		expect(orgs.shows(board('elsewhere'), 'acme')).toBe(false);
	});

	it('remembers the last organization opened while the user is in it', () => {
		orgs.remember('acme');
		expect(orgs.last).toBe('acme');
		orgs.remember('gone');
		expect(orgs.last).toBeUndefined();
		orgs.remember(undefined);
		expect(orgs.last).toBeUndefined();
	});

	it('loads the list once until refreshed', async () => {
		list.mockClear();
		await Promise.all([orgs.ensure(), orgs.ensure()]);
		expect(list).not.toHaveBeenCalled();
		await orgs.refresh();
		expect(list).toHaveBeenCalledTimes(1);
	});
});
