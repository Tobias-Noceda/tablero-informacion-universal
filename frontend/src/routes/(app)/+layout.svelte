<script lang="ts">
	import { m } from '$lib/paraglide/messages';

	import { onMount } from 'svelte';
	import { locales, getLocale, setLocale } from '$lib/paraglide/runtime';
	import logo from '$assets/logo.png';

	import { SvelteFlowProvider } from '@xyflow/svelte';

	import { session } from '$modules/session.svelte';
	import { setSidebar } from '$stores/sidebar';
	import { boardList, refreshBoards } from '$stores/boards';
	import { orgs } from '$stores/org.svelte';
	import { cn } from '$lib/utils';
	import { resolve } from '$app/paths';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import Dropdown from '$components/Dropdown/Dropdown.svelte';
	import DnDProvider from '$providers/DnDProvider.svelte';

	import Avatar from '$components/Avatar/Avatar.svelte';
	import Icon from '$components/Icon/Icon.svelte';
	import LocaleSwitcher from '$components/LocaleSwitcher/LocaleSwitcher.svelte';
	import OrgSwitcher from '$components/OrgSwitcher/OrgSwitcher.svelte';
	import { clickOutside } from '$lib/actions/clickOutside';

	onMount(() => {
		refreshBoards().catch((err) => console.error('Failed to load boards', err));
		orgs.refresh().catch((err) => console.error('Failed to load organizations', err));
	});

	const visibleBoards = $derived($boardList.filter((board) => orgs.shows(board)));

	// Signing out in another tab, or a session that could not be renewed.
	$effect(() => {
		if (session.status === 'anonymous') void goto(resolve('/login'));
	});

	let { children } = $props();

	const currentBoardId = $derived(page.params.id ?? '');

	let userDataOpen = $state<boolean>(false);
	let clickedOutside = $state<boolean>(false);

	// The layout's effect sends an anonymous session to /login.
	function logout() {
		userDataOpen = false;
		void session.logout();
	}

	const dropdownClass = cn(
		'absolute z-50 mt-0.5 w-fit flex-col',
		'bg-main-hover border border-main-hover rounded shadow-lg',
		'max-h-60 overflow-y-auto',
		// Custom scrollbar styles
		'scrollbar-thin scrollbar-thumb-tertiary scrollbar-track-transparent',
		'[&::-webkit-scrollbar]:w-2',
		'[&::-webkit-scrollbar-track]:bg-transparent',
		'[&::-webkit-scrollbar-thumb]:bg-tertiary',
		'[&::-webkit-scrollbar-thumb]:rounded-full',
		'[&::-webkit-scrollbar-thumb]:hover:bg-tertiary-hover'
	);

	const dropdownOptionClass = (isSelected: boolean) =>
		cn(
			'w-full justify-start flex items-center',
			'px-3 py-2.5 cursor-pointer transition-colors text-main-text',
			'bg-sidebar hover:bg-sidebar-hover',
			isSelected ? 'bg-sidebar-hover font-medium' : '',
			'overflow-x-hidden'
		);

	const dropdownSectionClass = cn(
		'px-3 py-2.5 font-medium text-main-text',
		'bg-sidebar-hover',
		'border-b border-main-border'
	);
</script>

<SvelteFlowProvider>
	<div class="flex flex-col h-screen w-screen">
		<div
			class="flex flex-row bg-main items-center justify-between border-b border-main-border px-2 py-1 z-100!"
		>
			<button
				class="flex flex-row items-center gap-4 cursor-pointer"
				onclick={() => {
					setSidebar(null);
					goto(resolve('/'));
				}}
			>
				<img src={logo} alt="logo" class="logo" />
			</button>
			<div class="flex flex-row items-center gap-3">
				<OrgSwitcher />
				<Dropdown
					placeholder={m['board_dropdown_placeholder']()}
					options={visibleBoards.map((board) => ({ label: board.name, value: board.id }))}
					value={currentBoardId}
					onchange={(value) => {
						if (value && value !== currentBoardId) {
							setSidebar(null);
							goto(resolve(`/board/${value}`));
						}
					}}
				/>
			</div>

			{#if session.user}
				<div class="relative flex w-fit flex-col gap-1">
					<button
						class={cn(
							'relative flex flex-row items-center gap-2',
							'px-4 py-2 rounded-md',
							'hover:bg-main-hover transition-colors',
							'text-main-text text-sm font-medium',
							'focus:outline-none focus:ring-none',
							'cursor-pointer',
							userDataOpen ? 'bg-main-hover' : ''
						)}
						aria-label={m['header.menu']()}
						aria-haspopup="menu"
						aria-expanded={userDataOpen}
						onclick={() => {
							if (clickedOutside) {
								clickedOutside = false;
								return;
							}
							userDataOpen = !userDataOpen;
						}}
					>
						<span class="max-w-40 truncate">{session.user.name}</span>
						<Avatar src={session.user.picture} class="w-6 h-6" />
						<Icon
							name={userDataOpen ? 'chevron-up' : 'chevron-down'}
							class="w-4 h-4 text-main-text"
						/>
					</button>

					{#if userDataOpen}
						<div
							role="menu"
							class={cn(dropdownClass, 'left-0 top-full w-full')}
							use:clickOutside={() => {
								clickedOutside = true;
								userDataOpen = false;
								setTimeout(() => (clickedOutside = false), 100);
							}}
						>
							<a
								role="menuitem"
								href={resolve('/profile')}
								class={dropdownOptionClass(false)}
								onclick={() => (userDataOpen = false)}
							>
								{m['header.profile']()}
							</a>
							<a
								role="menuitem"
								href={resolve('/orgs')}
								class={cn(dropdownOptionClass(false), 'border-b border-main-border')}
								onclick={() => (userDataOpen = false)}
							>
								{m['orgs.menu']()}
							</a>
							<div class={dropdownSectionClass}>{m['user_info.language.title']()}</div>
							{#each locales as locale (locale)}
								<button
									role="menuitem"
									class={dropdownOptionClass(getLocale() === locale)}
									onclick={() => {
										setLocale(locale);
										userDataOpen = false;
									}}
								>
									{m[`user_info.language.${locale}`]()}
								</button>
							{/each}
							<button
								role="menuitem"
								class={cn(dropdownOptionClass(false), 'border-t border-main-border')}
								onclick={logout}
							>
								<Icon name="logout" class="w-5 h-5 text-main-text mb-0.5 mr-1" />
								{m['user_info.logout']()}
							</button>
						</div>
					{/if}
				</div>
			{:else}
				<LocaleSwitcher />
			{/if}
		</div>
		<div class="flex flex-col flex-1 min-h-0 overflow-auto bg-background">
			<!-- Pages in (app) may assume a user; signing out unmounts them before it navigates. -->
			{#if session.user}
				<DnDProvider>
					{@render children()}
				</DnDProvider>
			{/if}
		</div>
	</div>
</SvelteFlowProvider>

<style>
	.logo {
		width: 180px;
		height: auto;
	}
</style>
