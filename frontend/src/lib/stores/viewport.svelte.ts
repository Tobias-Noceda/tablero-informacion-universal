import { MediaQuery } from 'svelte/reactivity';

// Below Tailwind's `md` breakpoint (768px) the app is read-only: boards can be
// looked at but not edited. Use this for behaviour only; layout that merely
// changes with the width belongs in `md:` classes.
export const mobile = new MediaQuery('(max-width: 767px)');
