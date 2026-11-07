// The app's own theme, kept in a cookie so it overrides the system's for this
// site only. No cookie: follow the system. src/app.html reads it before the
// first paint (keep the name in sync there).
export const THEME_COOKIE = 'tiu-theme';
export const THEME_MAX_AGE = 60 * 60 * 24 * 365;

export type ThemePreference = 'system' | 'light' | 'dark';

export function parseTheme(cookie: string): ThemePreference {
	for (const part of cookie.split(';')) {
		const [name, value] = part.trim().split('=');
		if (name === THEME_COOKIE && (value === 'light' || value === 'dark')) return value;
	}
	return 'system';
}

// The Set-Cookie string for a preference; "system" deletes the cookie.
export function themeCookie(preference: ThemePreference, secure: boolean): string {
	const attributes = `Path=/; SameSite=Lax${secure ? '; Secure' : ''}`;
	return preference === 'system'
		? `${THEME_COOKIE}=; Max-Age=0; ${attributes}`
		: `${THEME_COOKIE}=${preference}; Max-Age=${THEME_MAX_AGE}; ${attributes}`;
}

// What turning dark mode on or off means: matching the system goes back to
// following it.
export function preferenceFor(dark: boolean, systemDark: boolean): ThemePreference {
	if (dark === systemDark) return 'system';
	return dark ? 'dark' : 'light';
}
