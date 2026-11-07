import { describe, expect, it } from 'vitest';
import { parseTheme, preferenceFor, THEME_MAX_AGE, themeCookie } from './theme';

describe('parseTheme', () => {
	it('reads the cookie among others', () => {
		expect(parseTheme('a=1; tiu-theme=dark; b=2')).toBe('dark');
		expect(parseTheme('tiu-theme=light')).toBe('light');
	});

	it('follows the system otherwise', () => {
		expect(parseTheme('')).toBe('system');
		expect(parseTheme('tiu-theme=blue')).toBe('system');
		expect(parseTheme('my-tiu-theme=dark')).toBe('system');
	});
});

describe('themeCookie', () => {
	it('stores a choice for a year', () => {
		expect(themeCookie('dark', true)).toBe(
			`tiu-theme=dark; Max-Age=${THEME_MAX_AGE}; Path=/; SameSite=Lax; Secure`
		);
	});

	it('deletes it to follow the system', () => {
		expect(themeCookie('system', false)).toBe('tiu-theme=; Max-Age=0; Path=/; SameSite=Lax');
	});
});

describe('preferenceFor', () => {
	it('only overrides what differs from the system', () => {
		expect(preferenceFor(true, false)).toBe('dark');
		expect(preferenceFor(false, true)).toBe('light');
		expect(preferenceFor(true, true)).toBe('system');
		expect(preferenceFor(false, false)).toBe('system');
	});
});
