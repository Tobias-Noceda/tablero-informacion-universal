import { type Locale } from '$lib/paraglide/runtime';
import { session } from '$modules/session.svelte';
import * as users from '$services/users';

// The language a signed-in user picks is also the one their mails arrive in,
// so it is saved first: switching reloads the page. Failing to save it must
// not keep the app in a language the user just turned away from.
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
