<script lang="ts">
	import { m } from '$lib/paraglide/messages';

	import { locales, getLocale, setLocale } from '$lib/paraglide/runtime';
	import logo from '$assets/logo.png';

	import { session } from '$modules/session.svelte';
	import { setSidebar } from '$stores/sidebar';
	import { boardList } from '$stores/boards';
	import { orgs } from '$stores/org.svelte';
	import { cn } from '$lib/utils';
	import { boardPath, landingPath } from '$lib/helpers/routes';
	import { resolve } from '$app/paths';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import Dropdown from '$components/Dropdown/Dropdown.svelte';

	import Avatar from '$components/Avatar/Avatar.svelte';
	import Icon from '$components/Icon/Icon.svelte';
	import LocaleSwitcher from '$components/LocaleSwitcher/LocaleSwitcher.svelte';
	import OrgSwitcher from '$components/OrgSwitcher/OrgSwitcher.svelte';
	import { clickOutside } from '$lib/actions/clickOutside';
	import { setCreatingBoard } from '$stores/creating';

	// The organization comes from the URL; none means Personal.
	const orgId = $derived(page.params.orgId);
	const currentBoardId = $derived(page.params.id ?? '');

	const visibleBoards = $derived($boardList.filter((board) => orgs.shows(board, orgId)));

	// The profile and organization management pages are about no board.
	const showsBoards = $derived(
		!['/(app)/profile', '/(app)/orgs'].some((route) => page.route.id?.startsWith(route))
	);

	let userDataOpen = $state<boolean>(false);
	let clickedOutside = $state<boolean>(false);

	// The layout's effect sends an anonymous session to /login.
	function logout() {
		userDataOpen = false;
		void session.logout();
	}

	const dropdownClass = cn(
		'absolute z-100 mt-0.5 w-fit flex-col',
		'bg-main-hover border border-main-hover rounded shadow-lg',
		// 'max-h-60 overflow-y-auto',
		// // Custom scrollbar styles
		// 'scrollbar-thin scrollbar-thumb-tertiary scrollbar-track-transparent',
		// '[&::-webkit-scrollbar]:w-2',
		// '[&::-webkit-scrollbar-track]:bg-transparent',
		// '[&::-webkit-scrollbar-thumb]:bg-tertiary',
		// '[&::-webkit-scrollbar-thumb]:rounded-full',
		// '[&::-webkit-scrollbar-thumb]:hover:bg-tertiary-hover'
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

	const addButtonClass = cn(
		'w-full justify-start flex items-center text-left',
		'px-3 py-2.5 cursor-pointer transition-colors text-main-text',
		'bg-sidebar hover:bg-sidebar-hover',
		'focus:outline-none focus:ring-none focus:bg-sidebar-hover',
		'overflow-x-hidden'
	);
</script>

<div
	class="flex flex-row bg-main items-center justify-between border-b border-main-border px-2 py-1 z-100!"
>
	<button
		class="flex flex-row items-center gap-4 cursor-pointer"
		onclick={() => {
			setSidebar(null);
			// Off a board or landing, back to the organization last opened.
			goto(resolve(landingPath(showsBoards ? orgId : orgs.last)));
		}}
	>
		<img src={logo} alt="logo" class="logo" />
	</button>
	<div class="flex flex-row items-center gap-3">
		{#if showsBoards}
			<OrgSwitcher />
			<Dropdown
				placeholder={m['board_dropdown_placeholder']()}
				options={visibleBoards.map((board) => ({ label: board.name, value: board.id }))}
				value={currentBoardId}
				onchange={(value) => {
					if (value && value !== currentBoardId) {
						setSidebar(null);
						goto(resolve(boardPath(value, orgId)));
					}
				}}
			>
				{#if orgId === undefined || orgs.byId(orgId)?.role === 'admin'}
					<button
						class={addButtonClass}
						onclick={() => setCreatingBoard(true)}
					>
						<Icon name="plus" class="w-5 h-5 text-main-text mb-0.5 mr-1" />
						<span class="min-w-0 flex-1 truncate">Create new</span>
					</button>
				{/if}
			</Dropdown>
		{/if}
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
				<Icon name={userDataOpen ? 'chevron-up' : 'chevron-down'} class="w-4 h-4 text-main-text" />
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

<style>
	.logo {
		width: 180px;
		height: auto;
	}
</style>
