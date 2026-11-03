import { type Locale } from '$lib/paraglide/runtime';
import { session } from '$modules/session.svelte';
import * as users from '$services/users';

// Saves the language a signed-in user picks, the one their mails arrive in.
// It does not switch the UI language (Paraglide's setLocale does, reloading).
// A failed save is only logged.
export async function switchLocale(locale: Locale) {
	const user = session.user;
	if (user && user.locale !== locale) {
		try {
			session.user = await users.update(user.id, { locale });
		} catch (err) {
			console.error('Could not save the language', err);
		}
	}
}
