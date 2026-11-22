<script lang="ts">
	import { resolve } from '$app/paths';
	import logo from '$assets/svgs/corcheria-m2-logo-fondo-oscuro.svg';
	import LocaleSwitcher from '$components/LocaleSwitcher/LocaleSwitcher.svelte';
	import { m } from '$lib/paraglide/messages';
	import { getLocale } from '$lib/paraglide/runtime';

	let { children } = $props();

	// The public pages have an address per language (they are plain HTML).
	const spanish = getLocale() === 'es';
	const about = spanish ? resolve('/acerca') : resolve('/about');
	const privacy = spanish ? resolve('/privacidad') : resolve('/privacy');
</script>

<div class="flex min-h-dvh flex-col bg-background">
	<header
		class="flex flex-row items-center justify-between border-b border-main-border bg-main px-4 py-1"
	>
		<a href={about}><img src={logo} alt="Corcheria Company" class="w-32 sm:w-44" /></a>
		<LocaleSwitcher />
	</header>
	<main class="flex flex-1 items-start justify-center p-6 sm:items-center">
		<div class="flex w-full max-w-sm flex-col gap-4 rounded-lg border border-border p-6 shadow-sm">
			{@render children()}
		</div>
	</main>
	<footer class="flex flex-row justify-center gap-4 px-4 py-4 text-sm">
		<a href={about} class="underline hover:text-main">Corcheria Company</a>
		<a href={privacy} class="underline hover:text-main">{m['legal.privacy_policy']()}</a
		>
	</footer>
</div>
