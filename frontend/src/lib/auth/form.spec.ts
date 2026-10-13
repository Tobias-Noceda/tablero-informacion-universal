import { describe, expect, it } from 'vitest';
import { AuthError } from '$services/auth';
import { m } from '$lib/paraglide/messages';
import { describeError, passwordError, safeNext } from './form';

describe('passwordError', () => {
	it('accepts what the backend accepts', () => {
		expect(passwordError('1234567')).toBe(m['auth.error_weak_password']());
		expect(passwordError('12345678')).toBeUndefined();
		expect(passwordError('x'.repeat(128))).toBeUndefined();
		expect(passwordError('x'.repeat(129))).toBe(m['auth.error_weak_password']());
	});
});

describe('describeError', () => {
	it('explains every code the auth routes answer', () => {
		const codes = [
			'invalid_credentials',
			'email_not_verified',
			'rate_limited',
			'weak_password',
			'invalid_email',
			'invalid_name',
			'invalid_token',
			'invalid_state',
			'google_not_configured',
			'provider_error'
		];
		for (const code of codes) {
			expect(describeError(new AuthError(code, 400))).not.toBe(m['auth.error_unknown']());
		}
	});

	it('falls back to a generic message', () => {
		expect(describeError(new AuthError('internal', 500))).toBe(m['auth.error_unknown']());
		expect(describeError(new TypeError('offline'))).toBe(m['auth.error_unknown']());
	});
});

describe('safeNext', () => {
	it('keeps paths on this site and drops the rest', () => {
		expect(safeNext('/board/1?x=2')).toBe('/board/1?x=2');
		for (const next of [
			null,
			'',
			'https://evil.example',
			'//evil.example',
			'/\\evil.example',
			'board'
		]) {
			expect(safeNext(next)).toBe('/');
		}
	});
});
