<script lang="ts">
	import { m } from '$lib/paraglide/messages';

	import { locales, getLocale, setLocale } from '$lib/paraglide/runtime';
	import logo from '$assets/svgs/corcheria-m2-logo-fondo-oscuro.svg';

	import { session } from '$modules/session.svelte';
	import { setSharing, setSidebar } from '$stores/sidebar';
	import { boardList } from '$stores/boards';
	import { orgs } from '$stores/org.svelte';
	import { cn } from '$lib/utils';
	import { boardPath, landingPath } from '$lib/helpers/routes';
	import { resolve } from '$app/paths';
	import { afterNavigate, goto } from '$app/navigation';
	import { page } from '$app/state';
	import Dropdown from '$components/Dropdown/Dropdown.svelte';

	import Avatar from '$components/Avatar/Avatar.svelte';
	import Icon from '$components/Icon/Icon.svelte';
	import LocaleSwitcher from '$components/LocaleSwitcher/LocaleSwitcher.svelte';
	import OrgSwitcher from '$components/OrgSwitcher/OrgSwitcher.svelte';
	import { clickOutside } from '$lib/actions/clickOutside';
	import { setCreatingBoard } from '$stores/creating';
	import { theme } from '$stores/theme.svelte';
	import Switch from '$components/Switch/Switch.svelte';
	import Drawer from '$components/Drawer/Drawer.svelte';
	import { mobile } from '$stores/viewport.svelte';

	// The organization comes from the URL; none means Personal.
	const orgId = $derived(page.params.orgId);
	const currentBoardId = $derived(page.params.id ?? '');
	// An open card covers the whole board on a phone; the flap stays out of its way.
	const cardOpen = $derived(!!page.params.nodeId);

	const visibleBoards = $derived($boardList.filter((board) => orgs.shows(board, orgId)));

	// The profile and organization management pages are about no board.
	const showsBoards = $derived(
		!['/(app)/profile', '/(app)/orgs'].some((route) => page.route.id?.startsWith(route))
	);

	let userDataOpen = $state<boolean>(false);
	let clickedOutside = $state<boolean>(false);

	// On a phone the org and board pickers drop down from the bar.
	let navOpen = $state(false);
	let barHeight = $state(0);

	afterNavigate(() => (navOpen = false));
	$effect(() => {
		if (!mobile.current) navOpen = false;
	});

	function selectBoard(value: string) {
		if (value && value !== currentBoardId) {
			setSidebar(null);
			goto(resolve(boardPath(value, orgId)));
		}
	}

	// The layout's effect sends an anonymous session to /login.
	function logout() {
		userDataOpen = false;
		void session.logout();
	}

	const dropdownClass = cn(
		'absolute z-100 mt-0.5 w-fit flex-col',
		'bg-main-hover border border-main-hover rounded shadow-lg'
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

<!-- Org and board pickers; on a phone they live in the drawer, read-only (no "Create new"). -->
{#snippet pickers(phone: boolean)}
	<OrgSwitcher fullWidth={phone} />
	<Dropdown
		class={phone ? 'w-full' : undefined}
		placeholder={m['board_dropdown_placeholder']()}
		options={visibleBoards.map((board) => ({ label: board.name, value: board.id }))}
		value={currentBoardId}
		onchange={selectBoard}
	>
		{#if !phone && (orgId === undefined || orgs.byId(orgId)?.role === 'admin')}
			<button class={addButtonClass} onclick={() => setCreatingBoard(true)}>
				<Icon name="plus" class="w-5 h-5 text-main-text mb-0.5 mr-1" />
				<span class="min-w-0 flex-1 truncate">Create new</span>
			</button>
		{/if}
	</Dropdown>
{/snippet}

<!-- The tab hanging under the bar that opens (and, at the drawer's foot, closes) the pickers on a phone. -->
{#snippet flap()}
	<button
		class={cn(
			'flex items-center justify-center rounded-b-lg border border-t-0 border-main-border bg-main',
			'px-4 py-0.5 text-main-text shadow-sm transition-colors hover:bg-main-hover cursor-pointer md:hidden'
		)}
		aria-label={m['header.navigation']()}
		aria-haspopup="dialog"
		aria-expanded={navOpen}
		onclick={() => {
			userDataOpen = false;
			navOpen = !navOpen;
		}}
	>
		<Icon name={navOpen ? 'chevron-up' : 'chevron-down'} class="h-4 w-4" />
	</button>
{/snippet}

<div
	class="relative flex flex-row bg-main items-center justify-between gap-2 border-b border-main-border px-2 py-1 z-100!"
	bind:clientHeight={barHeight}
>
	<button
		class="flex shrink-0 flex-row items-center gap-4 cursor-pointer"
		onclick={() => {
			setSidebar(null);
			// Off a board or landing, back to the organization last opened.
			goto(resolve(landingPath(showsBoards ? orgId : orgs.last)));
		}}
	>
		<img src={logo} alt="Corcheria Company" class="h-auto w-28 md:w-45" />
	</button>
	<div class="flex flex-row items-center gap-3">
		{#if showsBoards}
			<div class="hidden flex-row items-center gap-3 md:flex">
				{@render pickers(false)}
			</div>
		{/if}
	</div>

	{#if session.user}
		<div class="flex min-w-0 flex-row items-center gap-1">
			<div class="relative flex w-fit min-w-0 max-w-[45vw] flex-col gap-1 md:max-w-none">
				<button
					class={cn(
						'relative flex min-w-0 flex-row items-center gap-1 md:gap-2',
						'px-2 py-1 md:px-4 md:py-2 rounded-md',
						'hover:bg-main-hover transition-colors',
						'text-main-text text-xs md:text-sm font-medium',
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
						navOpen = false;
						userDataOpen = !userDataOpen;
					}}
				>
					<span class="min-w-0 max-w-40 truncate">{session.user.name}</span>
					<Avatar src={session.user.picture} class="h-5 w-5 shrink-0 md:h-6 md:w-6" />
					<Icon
						name={userDataOpen ? 'chevron-up' : 'chevron-down'}
						class="h-3 w-3 shrink-0 text-main-text md:h-4 md:w-4"
					/>
				</button>

				{#if userDataOpen}
					<div
						role="menu"
						class={cn(dropdownClass, 'right-0 top-full w-full min-w-48')}
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
						<!-- The app's own theme; it starts as (and can go back to) the system's. -->
						<div
							role="none"
							class={cn(
								'flex w-full items-center justify-between gap-3',
								'px-3 py-2.5 text-main-text bg-sidebar',
								'border-t border-main-border'
							)}
						>
							<div class="flex min-w-0 flex-col">
								<span class="truncate text-sm font-medium">
									{m[`user_info.theme.${theme.dark ? 'dark' : 'light'}`]()}
								</span>
							</div>
							<!-- <Icon name="moon" class='w-4 h-4 text-black' /> -->
							<Switch
								ariaLabel={m['user_info.theme.title']()}
								checked={theme.dark}
								onchange={theme.setDark}
								innerClass="focus-visible:outline-main-text"
							>
								{#snippet checkedIcon()}
									<Icon name="moon" class="w-4 h-4 text-black" />
								{/snippet}
								{#snippet uncheckedIcon()}
									<Icon name="sun" class="w-4 h-4 text-black" />
								{/snippet}
							</Switch>
						</div>
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
		</div>
	{:else}
		<LocaleSwitcher />
	{/if}

	{#if session.user && showsBoards && !navOpen && !cardOpen}
		<div class="absolute top-full left-1/2 -translate-x-1/2 md:hidden">
			{@render flap()}
		</div>
	{/if}

	<Drawer
		open={navOpen}
		onclose={() => (navOpen = false)}
		top={barHeight}
		label={m['header.navigation']()}
		handle={flap}
	>
		{@render pickers(true)}
		{#if currentBoardId}
			<button
				class={cn(
					'flex w-full items-center gap-2 rounded-xl border border-main-border bg-main-hover',
					'px-3 py-2.5 text-main-text transition-colors hover:border-sidebar cursor-pointer'
				)}
				onclick={() => {
					navOpen = false;
					setSharing(true);
				}}
			>
				<Icon name="users" class="h-5 w-5" />
				<span class="min-w-0 flex-1 truncate text-start">{m['header.members']()}</span>
			</button>
		{/if}
	</Drawer>
</div>
