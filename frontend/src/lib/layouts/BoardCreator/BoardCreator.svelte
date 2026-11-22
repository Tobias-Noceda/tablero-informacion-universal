<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	
    import { boardPath } from '$lib/helpers/routes';
	
    import { m } from '$lib/paraglide/messages';

    import { isCreatingBoard, setCreatingBoard } from "$stores/creating";
	import { boardList, refreshBoards } from '$stores/boards';
	import { boardSpace } from '$stores/plan.svelte';
	import { orgs } from '$stores/org.svelte';
	
    import * as boardApi from '$services/board';
	import * as postItsApi from '$services/post-it';
	import * as edgesApi from '$services/edge';
	import { fillStarter, STARTERS, type StarterCard, type StarterKey } from '$lib/boards/starters';
	import { analyzeTitle } from '$lib/title';
	import { wellKnowns } from '$stores/well-knowns.svelte';
	import { toast } from '$stores/toast.svelte';
	import { cn } from '$lib/utils';
	
    import Modal from "$components/Modal/Modal.svelte";
	import Input from '$components/Input/Input.svelte';

    let boardName = $state('');
	// An example board to start from, or none.
	let starter = $state<StarterKey | null>(null);
	let creating = $state(false);

    const { org }: { org?: { id: string; name: string } } = $props();

	const starterNames: Record<StarterKey, () => string> = {
		finance: m['starters.finance'],
		trip: m['starters.trip'],
		dev: m['starters.dev']
	};
	const starterHints: Record<StarterKey, () => string> = {
		finance: m['starters.finance_hint'],
		trip: m['starters.trip_hint'],
		dev: m['starters.dev_hint']
	};
	const noteTexts: Record<string, () => string> = {
		'starters.finance_note_title': m['starters.finance_note_title'],
		'starters.finance_note': m['starters.finance_note'],
		'starters.trip_note_title': m['starters.trip_note_title'],
		'starters.trip_note': m['starters.trip_note'],
		'starters.dev_note_title': m['starters.dev_note_title'],
		'starters.dev_note': m['starters.dev_note']
	};

	function pickStarter(key: StarterKey | null) {
		// A name typed by the person stays; an untouched one follows the example.
		const previous = starter ? starterNames[starter]() : '';
		if (!boardName.trim() || boardName === previous) boardName = key ? starterNames[key]() : '';
		starter = key;
	}

	function titleOf(card: StarterCard) {
		if (card.note) return { text: noteTexts[card.note.title](), vars: false };
		const text = wellKnowns.title(card.wellKnown);
		const outputs = (wellKnowns.byKey(card.wellKnown)?.outputs ?? []).map((o) => o.key);
		return { text, vars: analyzeTitle(text, outputs).vars };
	}

	async function fill(boardId: string, key: StarterKey) {
		const chosen = STARTERS.find((s) => s.key === key);
		if (!chosen) return;
		await wellKnowns.ensure().catch(() => {});
		const { failed } = await fillStarter(
			chosen,
			{
				create: (wk, title, params) => postItsApi.create_well_known(boardId, title, wk, params),
				move: (id, x, y) => postItsApi.move(id, x, y),
				style: (id, color) => postItsApi.style(id, { color }),
				connect: (source, target) => edgesApi.connect(boardId, source, target)
			},
			titleOf,
			(text) => noteTexts[text]()
		);
		if (failed > 0) toast.notify({ title: m['starters.partial'](), variant: 'destructive' });
	}

	// Never opens at the plan's board limit, whatever asked for it.
	const space = $derived(boardSpace(org?.id, $boardList));

	// The creator stays mounted across pages, so a closed one starts empty
	// the next time it opens.
	function close() {
		boardName = '';
		starter = null;
		creating = false;
		setCreatingBoard(false);
	}

	async function handleCreate() {
		const trimmed = boardName.trim();
		if (!trimmed || creating) return;
		creating = true;
		const board = await boardApi.create(trimmed, org?.id);
		// Refused (a limit reached from another tab): the notice says why.
		if (!board?.id) {
			close();
			return;
		}
		if (starter) await fill(board.id, starter);
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
		acceptDisabled={!boardName.trim() || creating}
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
		<fieldset class="flex flex-col gap-2">
			<legend class="text-sm font-medium">{m['starters.label']()}</legend>
			<div class="grid grid-cols-2 gap-2" role="radiogroup" aria-label={m['starters.label']()}>
				{#each [null, ...STARTERS.map((s) => s.key)] as key (key ?? 'empty')}
					<button
						type="button"
						role="radio"
						aria-checked={starter === key}
						class={cn(
							'flex cursor-pointer flex-col items-start gap-0.5 rounded-md border p-2 text-left transition-colors',
							starter === key ? 'border-main bg-main-hover/20' : 'border-border hover:border-sidebar'
						)}
						onclick={() => pickStarter(key)}
					>
						<span class="text-sm font-semibold">{key ? starterNames[key]() : m['starters.empty']()}</span>
						<span class="text-xs opacity-70">{key ? starterHints[key]() : m['starters.empty_hint']()}</span>
					</button>
				{/each}
			</div>
		</fieldset>
		{#if org}
			<p class="text-xs opacity-70">{m['orgs.board_in']({ org: org.name })}</p>
		{/if}
	</Modal>
{/if}