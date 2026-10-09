import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { Board, Org } from '$types/api';

const acme: Org = { id: 'acme', name: 'Acme', members: [], created_at: '', role: 'member' };

vi.mock('$services/orgs', () => ({ list: vi.fn(async () => [acme]) }));

const { orgs, PERSONAL } = await import('./org.svelte');

function board(org?: string): Board {
	return { id: 'b', name: 'B', owner: 'o', members: [], org, postits: [], strands: [], envs: [] };
}

describe('organization selection', () => {
	beforeEach(async () => {
		orgs.select(PERSONAL);
		await orgs.refresh();
	});

	it('shows personal boards under Personal', () => {
		expect(orgs.shows(board())).toBe(true);
		expect(orgs.shows(board('acme'))).toBe(false);
	});

	it('shows only the selected organization’s boards', () => {
		orgs.select('acme');
		expect(orgs.current?.name).toBe('Acme');
		expect(orgs.shows(board('acme'))).toBe(true);
		expect(orgs.shows(board())).toBe(false);
	});

	it('files a board shared from an organization the user is not in as personal', () => {
		expect(orgs.shows(board('elsewhere'))).toBe(true);
		orgs.select('acme');
		expect(orgs.shows(board('elsewhere'))).toBe(false);
	});

	it('falls back to Personal when the selected organization is gone', async () => {
		orgs.select('gone');
		await orgs.refresh();
		expect(orgs.selected).toBe(PERSONAL);
		expect(orgs.current).toBeUndefined();
	});
});
