import { parseTheme, preferenceFor, themeCookie, type ThemePreference } from '$lib/helpers/theme';

const browser = typeof window !== 'undefined';

// Dark or light for this app, over the system's (see $lib/helpers/theme).
// The CSS reads <html data-theme>; src/app.html sets it before the first
// paint, this keeps it in sync afterwards.
class Theme {
	preference = $state<ThemePreference>(browser ? parseTheme(document.cookie) : 'system');
	systemDark = $state(false);

	dark = $derived(this.preference === 'system' ? this.systemDark : this.preference === 'dark');
	// What SvelteFlow's colorMode takes.
	colorMode = $derived(this.preference);

	constructor() {
		if (!browser) return;
		const query = window.matchMedia('(prefers-color-scheme: dark)');
		this.systemDark = query.matches;
		query.addEventListener('change', (e) => (this.systemDark = e.matches));
	}

	setDark = (dark: boolean) => {
		this.preference = preferenceFor(dark, this.systemDark);
		document.cookie = themeCookie(this.preference, location.protocol === 'https:');
		const root = document.documentElement;
		if (this.preference === 'system') delete root.dataset.theme;
		else root.dataset.theme = this.preference;
	};
}

export const theme = new Theme();
