<script lang="ts">
	import Button from '$components/Button/Button.svelte';
	import Input from '$components/Input/Input.svelte';
	import Modal from '$components/Modal/Modal.svelte';
	import { m } from '$lib/paraglide/messages';
	import type { BoardRole } from '$types/api';

	let {
		boardName,
        role,
		onclose,
        onedit,
		ondelete,
	}: { boardName: string; role: BoardRole; onclose: () => void; onedit: (name: string) => void; ondelete: () => void } = $props();

    let newName = $derived(boardName);

    let isDeleting = $state(false);
</script>

<Modal {onclose} onaccept={() => isDeleting = true} acceptText={m['board.delete']()} acceptClass="bg-red-600! hover:bg-red-700!">
	<h2 class="text-lg font-semibold">{m['board.title']()}</h2>

	{#if role !== 'owner'}
		<p class="text-xs opacity-70">{m['board.viewer_hint']()}</p>
	{:else}
		<form class="flex flex-col gap-2 border-t border-main-border pt-3" onsubmit={() => onedit(newName)}>
			<Input
				label={m['board.title_label']()}
				type="text"
				autocomplete="off"
				bind:value={boardName}
				required
			/>
            <Button type="submit" variant="primary">{m['board.modify_title']()}</Button>
		</form>
	{/if}
</Modal>

{#if isDeleting}
    <Modal onclose={() => isDeleting = false} onaccept={ondelete} acceptText={m['board.delete']()} acceptClass="bg-red-600! hover:bg-red-700!">
        <h2 class="text-lg font-semibold">{m['board.delete']()}</h2>
        <p class="text-xs opacity-70">{m['board.delete_confirm']({ name: boardName })}</p>
    </Modal>
{/if}
