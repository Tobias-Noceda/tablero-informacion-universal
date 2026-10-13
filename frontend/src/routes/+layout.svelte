<script lang="ts">
	import { m } from '$lib/paraglide/messages';

	import { onMount } from 'svelte';
	import { locales, type Locale } from '$lib/paraglide/runtime';
	import './layout.css';
	import favicon from '$assets/favicon.svg';
	import logo from '$assets/logo.png';

	import { SvelteFlowProvider } from '@xyflow/svelte';

	import { setLocale, getLocale } from '$lib/paraglide/runtime';
	import { setSidebar } from '$stores/sidebar';
	import { boardList, refreshBoards } from '$stores/boards';
	import { cn } from '$lib/utils';
	import { resolve } from '$app/paths';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import Dropdown from '$components/Dropdown/Dropdown.svelte';
	import DnDProvider from '$providers/DnDProvider.svelte';

	import { getUser } from '$stores/user';
	import Avatar from '$components/Avatar/Avatar.svelte';
	import Icon from '$components/Icon/Icon.svelte';
	import { clickOutside } from '$lib/actions/clickOutside';

	onMount(() => {
		refreshBoards().catch((err) => console.error('Failed to load boards', err));
	});

	let { children } = $props();

	// const buttonClass = $derived(cn(
	// 	"flex p-3 rounded-md text-white hover:bg-main-hover transition-colors h-fit",
	// 	$getSidebar ? "bg-main-hover" : "",
	// 	"cursor-pointer"
	// ));

	const languageClass = (locale: Locale) =>
		cn(
			'px-4 py-2 rounded-md text-main-text hover:bg-main-hover transition-colors text-main-text text-sm font-medium',
			getLocale() === locale ? 'bg-main-hover' : '',
			'cursor-pointer'
		);

	const currentBoardId = $derived(page.params.id ?? '');

	let userDataOpen = $state<boolean>(false);
	let clickedOutside = $state<boolean>(false);

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

<svelte:head><link rel="icon" href={favicon} /></svelte:head>

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
				<!-- <button onclick={() => toggleSidebar()} class={buttonClass}>
					<Icon name="menu" class="w-5 h-5 text-white" />
				</button> -->
				<img src={logo} alt="logo" class="logo" />
			</button>
			<div class="flex flex-row items-center gap-3">
				<Dropdown
					placeholder={m['board_dropdown_placeholder']()}
					options={$boardList.map((board) => ({ label: board.name, value: board.id }))}
					value={currentBoardId}
					onchange={(value) => {
						if (value && value !== currentBoardId) {
							setSidebar(null);
							goto(resolve(`/board/${value}`));
						}
					}}
				/>
			</div>

			{#if $getUser}
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
						onclick={() => {
							if (clickedOutside) {
								clickedOutside = false;
								return;
							}
							userDataOpen = !userDataOpen;
						}}
					>
						{$getUser.username}
						<Avatar src={$getUser.avatarUrl} class="w-6 h-6" />
						<Icon
							name={userDataOpen ? 'chevron-up' : 'chevron-down'}
							class="w-4 h-4 text-main-text"
						/>
					</button>

					{#if userDataOpen}
						<div
							class={cn(dropdownClass, 'left-0 top-full w-full')}
							use:clickOutside={() => {
								clickedOutside = true;
								userDataOpen = false;
								setTimeout(() => (clickedOutside = false), 100);
							}}
						>
							<div class={dropdownSectionClass}>{m['user_info.language.title']()}</div>
							{#each locales as locale (locale)}
								<button
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
								class={cn(dropdownOptionClass(false), 'border-t border-main-border')}
								onclick={() => {
									// TODO: Implement logout functionality
									userDataOpen = false;
								}}
							>
								<Icon name="logout" class="w-5 h-5 text-main-text mb-0.5 mr-1" />
								{m['user_info.logout']()}
							</button>
						</div>
					{/if}
				</div>
			{:else}
				<div class="flex flex-row items-center w-fit gap-3 mr-1">
					{#each locales as locale (locale)}
						<button class={languageClass(locale)} onclick={() => setLocale(locale)}
							>{locale.toUpperCase()}</button
						>
					{/each}
				</div>
			{/if}
		</div>
		<div class="flex flex-row h-full w-screen">
			<DnDProvider>
				{@render children()}
			</DnDProvider>
		</div>
	</div>
</SvelteFlowProvider>

<style>
	.logo {
		width: 180px;
		height: auto;
	}
</style>
