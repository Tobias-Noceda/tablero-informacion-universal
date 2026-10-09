import { resolve } from '$app/paths';
import type { ResolvedPathname } from '$app/types';
import { m } from '$lib/paraglide/messages';
import { AuthError } from '$services/auth';

// Same bounds the backend enforces, checked before a round trip.
export const PASSWORD_MIN = 8;
export const PASSWORD_MAX = 128;

export function passwordError(password: string) {
	if (password.length < PASSWORD_MIN || password.length > PASSWORD_MAX) {
		return m['auth.error_weak_password']();
	}
}

const messages: Record<string, () => string> = {
	invalid_credentials: m['auth.error_invalid_credentials'],
	email_not_verified: m['auth.error_email_not_verified'],
	rate_limited: m['auth.error_rate_limited'],
	weak_password: m['auth.error_weak_password'],
	invalid_email: m['auth.error_invalid_email'],
	invalid_name: m['auth.error_invalid_name'],
	invalid_token: m['auth.error_invalid_token'],
	invalid_state: m['auth.error_invalid_state'],
	google_not_configured: m['auth.error_google_not_configured'],
	provider_error: m['auth.error_provider_error']
};

export function describeError(err: unknown) {
	const message = err instanceof AuthError ? messages[err.code] : undefined;
	return (message ?? m['auth.error_unknown'])();
}

// Where to land after signing in: only a path on this site, so a crafted
// ?next= can never send a fresh session somewhere else. It was read from the
// address bar, so it already carries the base path.
export function safeNext(next: string | null): ResolvedPathname {
	if (!next || !next.startsWith('/') || next.startsWith('//') || next.includes('\\')) {
		return resolve('/');
	}
	return next as ResolvedPathname;
}
