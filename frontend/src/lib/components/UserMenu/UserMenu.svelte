<script lang="ts">
	import { resolve } from '$app/paths';
	import { m } from '$lib/paraglide/messages';
	import { session } from '$modules/session.svelte';

	let open = $state(false);
	let menu: HTMLDivElement | undefined = $state();

	const user = $derived(session.user);
	const initial = $derived(user?.name.trim().charAt(0).toUpperCase() || '?');

	function closeOnOutsideClick(event: MouseEvent) {
		if (open && menu && !menu.contains(event.target as Node)) open = false;
	}

	// The app layout sends an anonymous session to /login.
	function logout() {
		open = false;
		void session.logout();
	}
</script>

<svelte:window onclick={closeOnOutsideClick} />

{#if user}
	<div class="relative" bind:this={menu}>
		<button
			type="button"
			class="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1 text-white transition-colors hover:bg-main-hover"
			aria-label={m['header.menu']()}
			aria-haspopup="menu"
			aria-expanded={open}
			onclick={() => (open = !open)}
		>
			{#if user.picture}
				<img src={user.picture} alt="" referrerpolicy="no-referrer" class="h-8 w-8 rounded-full" />
			{:else}
				<span
					class="flex h-8 w-8 items-center justify-center rounded-full bg-sidebar font-semibold"
				>
					{initial}
				</span>
			{/if}
			<span class="max-w-40 truncate text-sm font-medium">{user.name}</span>
		</button>

		{#if open}
			<div
				role="menu"
				class="absolute right-0 mt-1 flex w-48 flex-col rounded-md border border-border bg-background py-1 text-foreground shadow-lg"
			>
				<a
					role="menuitem"
					href={resolve('/profile')}
					class="px-4 py-2 text-sm hover:bg-skeleton"
					onclick={() => (open = false)}
				>
					{m['header.profile']()}
				</a>
				<a
					role="menuitem"
					href={resolve('/orgs')}
					class="px-4 py-2 text-sm hover:bg-skeleton"
					onclick={() => (open = false)}
				>
					{m['orgs.menu']()}
				</a>
				<button
					type="button"
					role="menuitem"
					class="cursor-pointer px-4 py-2 text-left text-sm hover:bg-skeleton"
					onclick={logout}
				>
					{m['header.logout']()}
				</button>
			</div>
		{/if}
	</div>
{/if}
