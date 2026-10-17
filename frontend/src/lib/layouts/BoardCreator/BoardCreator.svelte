<script>
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	
    import { boardPath } from '$lib/helpers/routes';
	
    import { m } from '$lib/paraglide/messages';

    import { isCreatingBoard, setCreatingBoard } from "$stores/creating";
	import { refreshBoards } from '$stores/boards';
	
    import * as boardApi from '$services/board';
	
    import Modal from "$components/Modal/Modal.svelte";
	import Input from '$components/Input/Input.svelte';

    let boardName = $state('');

    const { org } = $props();

	async function handleCreate() {
		const trimmed = boardName.trim();
		if (!trimmed) return;
		const board = await boardApi.create(trimmed, org?.id);
		await refreshBoards();
		setCreatingBoard(false);
		await goto(resolve(boardPath(board.id, org?.id)));
	}
</script>

{#if $isCreatingBoard}
	<Modal
		onclose={() => setCreatingBoard(false)}
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