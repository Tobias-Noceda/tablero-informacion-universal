<script lang="ts">
	import Icon from "$components/Icon/Icon.svelte";
	import Input from "$components/Input/Input.svelte";
	import Modal from "$components/Modal/Modal.svelte";
	import { m } from "$lib/paraglide/messages";

	let showModal = $state(false);
    let boardName = $state('');

	function handleCreate() {
        const trimmed = boardName.trim();
        if (!trimmed) return;
        // TODO: call API to create board
		console.log('Create board:', trimmed);
        showModal = false;
	}
</script>

<div class="flex flex-1 items-center justify-center">
    <button
		class="flex h-fit w-fit items-center justify-center gap-2 hover:text-primary! hover:underline cursor-pointer"
		onclick={() => showModal = true}
	>
        <Icon name="plus" class="w-10 h-10" />
        <p class="text-secondary text-lg font-medium m-0">{m.create_board()}</p>
    </button>
</div>

{#if showModal}
	<Modal
        onclose={() => showModal = false}
        onaccept={handleCreate}
        acceptDisabled={!boardName.trim()}
        acceptText={m.create()}
    >
        <h2 class="text-lg font-semibold">{m.create_board()}</h2>
        <Input
            label={m.board_name_label()}
            placeholder={m.board_name_placeholder()}
            bind:value={boardName}
            onsubmit={handleCreate}
            required
        />
    </Modal>
{/if}
