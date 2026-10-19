import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { Profile } from '$types/api';

const setLocale = vi.hoisted(() => vi.fn());
const update = vi.hoisted(() => vi.fn());
const session = vi.hoisted(() => ({ user: null as Profile | null }));

vi.mock('$lib/paraglide/runtime', () => ({ setLocale }));
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
		setLocale.mockReset();
		update.mockReset();
		session.user = null;
	});

	it('saves the language of a signed-in user before switching', async () => {
		session.user = ana;
		update.mockResolvedValue({ ...ana, locale: 'es' });

		await switchLocale('es');

		expect(update).toHaveBeenCalledWith(ana.id, { locale: 'es' });
		expect(session.user?.locale).toBe('es');
		expect(setLocale).toHaveBeenCalledWith('es');
		expect(update.mock.invocationCallOrder[0]).toBeLessThan(setLocale.mock.invocationCallOrder[0]);
	});

	it('only switches for someone signed out', async () => {
		await switchLocale('es');

		expect(update).not.toHaveBeenCalled();
		expect(setLocale).toHaveBeenCalledWith('es');
	});

	it('switches anyway when saving fails', async () => {
		session.user = ana;
		update.mockRejectedValue(new Error('offline'));
		vi.spyOn(console, 'error').mockImplementation(() => {});

		await switchLocale('es');

		expect(setLocale).toHaveBeenCalledWith('es');
		expect(session.user?.locale).toBe('en');
	});

	it('does not save the language the user already has', async () => {
		session.user = { ...ana, locale: 'es' };

		await switchLocale('es');

		expect(update).not.toHaveBeenCalled();
		expect(setLocale).toHaveBeenCalledWith('es');
	});
});
