<script lang="ts">
	import { resolve } from '$app/paths';
	import logo from '$assets/svgs/corcheria-m2-logo-fondo-oscuro.svg';
	import LocaleSwitcher from '$components/LocaleSwitcher/LocaleSwitcher.svelte';

	let { children } = $props();
</script>

<div class="flex min-h-screen flex-col bg-background">
	<header
		class="flex flex-row items-center justify-between border-b border-main-border bg-main px-4 py-1"
	>
		<a href={resolve('/about')}><img src={logo} alt="Corcheria Company" class="w-44" /></a>
		<LocaleSwitcher />
	</header>
	<main class="flex flex-1 items-start justify-center p-6 sm:items-center">
		<div class="flex w-full max-w-sm flex-col gap-4 rounded-lg border border-border p-6 shadow-sm">
			{@render children()}
		</div>
	</main>
	<footer class="flex flex-row justify-center gap-4 px-4 py-4 text-sm">
		<a href={resolve('/about')} class="underline hover:text-main">Corcheria Company</a>
		<a href={resolve('/privacy')} class="underline hover:text-main">Privacy Policy</a>
	</footer>
</div>
