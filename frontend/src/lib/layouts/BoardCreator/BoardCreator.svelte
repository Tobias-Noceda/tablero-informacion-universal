<script>
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	
    import { boardPath } from '$lib/helpers/routes';
	
    import { m } from '$lib/paraglide/messages';

    import { isCreatingBoard, setCreatingBoard } from "$stores/creating";
	import { boardList, refreshBoards } from '$stores/boards';
	import { boardSpace } from '$stores/plan.svelte';
	import { orgs } from '$stores/org.svelte';
	
    import * as boardApi from '$services/board';
	
    import Modal from "$components/Modal/Modal.svelte";
	import Input from '$components/Input/Input.svelte';

    let boardName = $state('');

    const { org } = $props();

	// Never opens at the plan's board limit, whatever asked for it.
	const space = $derived(boardSpace(org?.id, $boardList));

	// The creator stays mounted across pages, so a closed one starts empty
	// the next time it opens.
	function close() {
		boardName = '';
		setCreatingBoard(false);
	}

	async function handleCreate() {
		const trimmed = boardName.trim();
		if (!trimmed) return;
		const board = await boardApi.create(trimmed, org?.id);
		// Refused (a limit reached from another tab): the notice says why.
		if (!board?.id) {
			close();
			return;
		}
		await refreshBoards();
		// The organization's board count moved.
		void orgs.refresh().catch(() => {});
		close();
		await goto(resolve(boardPath(board.id, org?.id)));
	}
</script>

{#if $isCreatingBoard && space.allowed}
	<Modal
		onclose={close}
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