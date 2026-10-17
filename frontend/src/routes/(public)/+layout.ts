// The home and privacy pages are read by people who are not signed in and by
// Google's OAuth review, which does not run scripts: they ship as plain HTML.
export const ssr = true;
export const prerender = true;
export const csr = false;
