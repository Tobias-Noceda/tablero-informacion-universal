import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { Board, Org } from '$types/api';

const { list } = vi.hoisted(() => {
	const acme: Org = { id: 'acme', name: 'Acme', members: [], created_at: '', role: 'member' };
	return { list: vi.fn(async () => [acme]) };
});
vi.mock('$services/orgs', () => ({ list }));

const session = vi.hoisted(() => {
	const listeners: ((id: string | null) => void)[] = [];
	return {
		user: { id: 'ana' } as { id: string } | null,
		onUserChange: (fn: (id: string | null) => void) => listeners.push(fn),
		switchTo(id: string | null) {
			listeners.forEach((fn) => fn(id));
			this.user = id ? { id } : null;
		}
	};
});
vi.mock('$modules/session.svelte', () => ({ session }));

const stored = new Map<string, string>();
vi.stubGlobal('localStorage', {
	getItem: (key: string) => stored.get(key) ?? null,
	setItem: (key: string, value: string) => void stored.set(key, value),
	removeItem: (key: string) => void stored.delete(key)
});

const { orgs } = await import('./org.svelte');

function board(org?: string): Board {
	return { id: 'b', name: 'B', owner: 'o', members: [], org, postits: [], strands: [], envs: [] };
}

describe('organizations', () => {
	beforeEach(async () => {
		stored.clear();
		session.switchTo('ana');
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

	it('forgets the previous user’s organizations when the user changes', async () => {
		expect(orgs.list).toHaveLength(1);
		list.mockClear();
		list.mockResolvedValueOnce([]);

		session.switchTo('bruno');
		expect(orgs.list).toEqual([]);
		await orgs.ensure();

		expect(list).toHaveBeenCalledTimes(1);
		expect(orgs.byId('acme')).toBeUndefined();
	});

	it('drops a list requested for the previous user', async () => {
		let release!: (orgs: Org[]) => void;
		list.mockImplementationOnce(() => new Promise<Org[]>((r) => (release = r)));
		const stale = orgs.refresh();

		session.switchTo('bruno');
		release([{ id: 'old', name: 'Old', members: [], created_at: '', role: 'admin' }]);
		await stale;

		expect(orgs.byId('old')).toBeUndefined();
	});

	it('remembers the last organization per user', async () => {
		orgs.remember('acme');
		session.switchTo('bruno');
		await orgs.ensure();
		expect(orgs.last).toBeUndefined();

		session.switchTo('ana');
		await orgs.ensure();
		expect(orgs.last).toBe('acme');
	});
});
