import type { BeforeSendEvent } from '@vercel/speed-insights/sveltekit';

// Speed Insights reports the address of every page it measures. Queries
// and fragments can carry one-time secrets (/verify?token=, /reset?token=,
// /oauth2/callback?code=&state=), so only the path leaves the browser; an
// address that cannot be read is not sent at all.
export function withoutQuery(event: BeforeSendEvent): BeforeSendEvent | null {
	try {
		const url = new URL(event.url);
		return { ...event, url: url.origin + url.pathname };
	} catch {
		return null;
	}
}
