import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { Profile } from '$types/api';

const update = vi.hoisted(() => vi.fn());
const session = vi.hoisted(() => ({ user: null as Profile | null }));

vi.mock('$services/users', () => ({ update }));
vi.mock('$modules/session.svelte', () => ({ session }));

const { switchLocale } = await import('./locale');

const ana: Profile = {
	id: '0b9a4c5e-1111-4222-8333-944445555666',
	email: 'ana@example.com',
	email_verified: true,
	name: 'Ana',
	locale: 'en',
	admin: false,
	identities: ['password'],
	created_at: '2026-10-04T00:00:00Z'
};

describe('switchLocale', () => {
	beforeEach(() => {
		update.mockReset();
		session.user = null;
	});

	it('saves the language of a signed-in user', async () => {
		session.user = ana;
		update.mockResolvedValue({ ...ana, locale: 'es' });

		await switchLocale('es');

		expect(update).toHaveBeenCalledWith(ana.id, { locale: 'es' });
		expect(session.user?.locale).toBe('es');
	});

	it('saves nothing for someone signed out', async () => {
		await switchLocale('es');

		expect(update).not.toHaveBeenCalled();
	});

	it('keeps the session as it was when saving fails', async () => {
		session.user = ana;
		update.mockRejectedValue(new Error('offline'));
		const error = vi.spyOn(console, 'error').mockImplementation(() => {});

		await expect(switchLocale('es')).resolves.toBeUndefined();

		expect(session.user?.locale).toBe('en');
		expect(error).toHaveBeenCalled();
		error.mockRestore();
	});

	it('does not save the language the user already has', async () => {
		session.user = { ...ana, locale: 'es' };

		await switchLocale('es');

		expect(update).not.toHaveBeenCalled();
	});
});
