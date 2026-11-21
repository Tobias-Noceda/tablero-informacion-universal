import { injectSpeedInsights } from '@vercel/speed-insights/sveltekit';
import { withoutQuery } from '$lib/helpers/speed-insights';

export const ssr = false;
export const prerender = false;

// Real visitors' load and interaction times, per route. Only a Vercel
// deployment serves the script (/_vercel/speed-insights/script.js); the
// Docker build and the dev server skip it.
if (__VERCEL_BUILD__) injectSpeedInsights({ beforeSend: withoutQuery });
