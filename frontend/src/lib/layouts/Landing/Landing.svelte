<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import BoardPreview from '$components/BoardPreview/BoardPreview.svelte';
	import Icon from '$components/Icon/Icon.svelte';
	import Input from '$components/Input/Input.svelte';
	import Modal from '$components/Modal/Modal.svelte';
	import { m } from '$lib/paraglide/messages';
	import { boardPath } from '$lib/helpers/routes';
	import * as boardApi from '$services/board';
	import { boardList, refreshBoards } from '$stores/boards';
	import { orgs } from '$stores/org.svelte';
	import type { BoardRole, Org } from '$types/api';

	// The organization whose boards this is, or none for Personal.
	let { org }: { org?: Org } = $props();

	const boards = $derived($boardList.filter((board) => orgs.shows(board, org?.id)));

	const roleLabel: Record<BoardRole, () => string> = {
		owner: m['members.role_owner'],
		editor: m['members.role_editor'],
		viewer: m['members.role_viewer']
	};

	let showModal = $state(false);
	let boardName = $state('');

	async function handleCreate() {
		const trimmed = boardName.trim();
		if (!trimmed) return;
		const board = await boardApi.create(trimmed, org?.id);
		await refreshBoards();
		showModal = false;
		await goto(resolve(boardPath(board.id, org?.id)));
	}
</script>

<div class="flex flex-1 flex-col gap-4 overflow-y-auto p-6">
	<h1 class="text-3xl font-semibold text-main-text">{org?.name ?? m['landing.title_personal']()}</h1>

	<div class="grid grid-cols-[repeat(auto-fill,minmax(25rem,1fr))] gap-5">
		{#each boards as board (board.id)}
			<!-- {#each [0, 1, 2, 3, 4, 5, 6, 7, 8, 9] as i (board.id + '-' + i)} -->
				<a
					href={resolve(boardPath(board.id, org?.id))}
					class="flex aspect-4/3 flex-col overflow-hidden rounded-lg border border-main-border bg-sidebar transition-colors hover:border-main-border-hover"
				>
					<div class="min-h-0 flex-1 p-2 bg-background">
						<BoardPreview {board} />
					</div>
					<div
						class="flex items-center justify-between gap-2 border-t border-main-border px-3 py-2"
					>
						<span class="truncate font-medium text-main-text">{board.name}</span>
						<span class="shrink-0 text-xs opacity-70">
							{m['landing.cards']({ count: board.postits.length })}
							{#if board.role}· {roleLabel[board.role]()}{/if}
						</span>
					</div>
				</a>
			<!-- {/each} -->
		{/each}
		<button
			class="flex aspect-4/3 cursor-pointer flex-col items-center justify-center gap-2 rounded-lg border-2 border-dashed border-main-border text-secondary transition-colors hover:border-main-border-hover hover:text-primary"
			onclick={() => (showModal = true)}
		>
			<Icon name="plus" class="h-10 w-10" />
			<span class="font-medium">{m['landing.create_board']()}</span>
		</button>
	</div>

	{#if boards.length === 0}
		<p class="text-sm opacity-70">{m['landing.empty']()}</p>
	{/if}
</div>

{#if showModal}
	<Modal
		onclose={() => (showModal = false)}
		onaccept={handleCreate}
		acceptDisabled={!boardName.trim()}
		acceptText={m['create']()}
	>
		<h2 class="text-lg font-semibold">{m['landing.create_board']()}</h2>
		<Input
			label={m.board_name_label()}
			placeholder={m.board_name_placeholder()}
			bind:value={boardName}
			onsubmit={handleCreate}
			required
		/>
		{#if org}
			<p class="text-xs opacity-70">{m['orgs.board_in']({ org: org.name })}</p>
		{/if}
	</Modal>
{/if}
