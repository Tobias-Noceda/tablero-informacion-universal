<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import logo from '$assets/svgs/corcheria-m2-logo-fondo-oscuro.svg';
	import { m } from '$lib/paraglide/messages';
	import { cn } from '$lib/utils';

	let { children } = $props();

	// These pages are prerendered HTML without scripts (see +layout.ts), so
	// the language is the address's, not the visitor's cookie: English at
	// /about and /privacy, Spanish at /acerca and /privacidad (Spanish words,
	// not an /es prefix: Paraglide's reroute strips locale prefixes).
	// Each public page and its translation, by route; resolve() turns them
	// into links (relative ones when prerendering, so match on the route).
	const PAGES = [
		{ en: '/about', es: '/acerca' },
		{ en: '/privacy', es: '/privacidad' }
	] as const;
	const current = $derived(
		PAGES.find((p) => p.en === page.url.pathname || p.es === page.url.pathname) ?? PAGES[0]
	);
	const spanish = $derived(current.es === page.url.pathname);
	const locale = $derived(spanish ? 'es' : 'en');
	const english = $derived(resolve(current.en));
	const castellano = $derived(resolve(current.es));
	const home = $derived(resolve(spanish ? PAGES[0].es : PAGES[0].en));
	const privacy = $derived(resolve(spanish ? PAGES[1].es : PAGES[1].en));

	const languageClass = (active: boolean) =>
		cn('rounded-md px-3 py-1.5 hover:bg-main-hover', active && 'bg-main-hover');
</script>

<svelte:head>
	<link rel="alternate" hreflang="en" href={english} />
	<link rel="alternate" hreflang="es" href={castellano} />
</svelte:head>

<div class="flex min-h-dvh flex-col bg-background" lang={locale}>
	<header
		class="flex flex-row items-center justify-between gap-2 sm:gap-4 border-b border-main-border bg-main px-4 py-1"
	>
		<a href={home}><img src={logo} alt="Corcheria Company" class="w-32 sm:w-44" /></a>
		<nav
			class="flex flex-row items-center gap-2 sm:gap-4 text-xs sm:text-sm font-medium text-main-text"
		>
			<a href={resolve('/login')} class="hover:underline">{m['auth.login']({}, { locale })}</a>
			<a href={resolve('/register')} class="hover:underline"
				>{m['auth.register_title']({}, { locale })}</a
			>
			<a
				href={english}
				hreflang="en"
				class={languageClass(!spanish)}
				aria-current={!spanish ? 'page' : undefined}>EN</a
			>
			<a
				href={castellano}
				hreflang="es"
				class={languageClass(spanish)}
				aria-current={spanish ? 'page' : undefined}>ES</a
			>
		</nav>
	</header>

	<main class="mx-auto flex w-full max-w-3xl flex-1 flex-col gap-6 px-4 py-10">
		{@render children()}
	</main>

	<footer
		class="flex flex-row flex-wrap items-center justify-center gap-4 border-t border-border px-4 py-4 text-sm"
	>
		<span>© 2026 Corcheria Company</span>
		<a href={home} class="underline hover:text-main">{m['legal.home']({}, { locale })}</a>
		<a href={privacy} class="underline hover:text-main"
			>{m['legal.privacy_policy']({}, { locale })}</a
		>
	</footer>
</div>
