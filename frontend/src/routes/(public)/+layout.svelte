<script lang="ts">
	import { resolve } from '$app/paths';
	import logo from '$assets/svgs/corcheria-m2-logo-fondo-oscuro.svg';

	let { children } = $props();
</script>

<div class="flex min-h-screen flex-col bg-background">
	<header
		class="flex flex-row items-center justify-between gap-4 border-b border-main-border bg-main px-4 py-1"
	>
		<a href={resolve('/about')}><img src={logo} alt="Corcheria Company" class="w-44" /></a>
		<nav class="flex flex-row items-center gap-4 text-sm font-medium text-main-text">
			<a href={resolve('/login')} class="hover:underline">Sign in</a>
			<a href={resolve('/register')} class="hover:underline">Create an account</a>
		</nav>
	</header>

	<main class="mx-auto flex w-full max-w-3xl flex-1 flex-col gap-6 px-4 py-10">
		{@render children()}
	</main>

	<footer
		class="flex flex-row flex-wrap items-center justify-center gap-4 border-t border-border px-4 py-4 text-sm"
	>
		<span>© 2026 Corcheria Company</span>
		<a href={resolve('/about')} class="underline hover:text-main">Home</a>
		<a href={resolve('/privacy')} class="underline hover:text-main">Privacy Policy</a>
	</footer>
</div>
