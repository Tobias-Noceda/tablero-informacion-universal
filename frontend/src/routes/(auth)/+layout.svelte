<script lang="ts">
	import logo from '$assets/logo.png';
	import LocaleSwitcher from '$components/LocaleSwitcher/LocaleSwitcher.svelte';

	let { children } = $props();
</script>

<div class="flex min-h-screen flex-col bg-background">
	<header
		class="flex flex-row items-center justify-between border-b border-main-border bg-main px-4 py-1"
	>
		<img src={logo} alt="logo" class="w-44" />
		<LocaleSwitcher />
	</header>
	<main class="flex flex-1 items-start justify-center p-6 sm:items-center">
		<div class="flex w-full max-w-sm flex-col gap-4 rounded-lg border border-border p-6 shadow-sm">
			{@render children()}
		</div>
	</main>
</div>
