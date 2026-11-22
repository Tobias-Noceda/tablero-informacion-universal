<script lang="ts">
  import '../index.css';

	import Icon from '$components/Icon/Icon.svelte';
	import { m } from '$lib/paraglide/messages';
	import { getLocale } from '$lib/paraglide/runtime';
	import { useDnD } from '$providers/DnDProvider.svelte';
	import { setManagingCardTypes } from '$stores/sidebar';
	import { wellKnowns } from '$stores/well-knowns.svelte';
	import { ONE_OFF } from '$lib/well-knowns/one-off';
	import SidebarElement from './SidebarElement.svelte';
	import { boardPlan } from '$stores/plan.svelte';
	import { canAddNode } from '$lib/billing/can';
	import PlanHint from '$components/PlanHint/PlanHint.svelte';
	import { matchesCard } from '$lib/well-knowns/search';

	const type = useDnD();

	// The catalogue comes from the backend; the board's load usually has it already.
	wellKnowns.ensure().catch(() => {});

	// At the plan's card limit there is nothing to drop, only a line saying so.
	const canAdd = $derived(canAddNode(boardPlan.limits, boardPlan.nodes));

	// What was typed in the search box narrows the card types listed.
	let search = $state('');
	const shown = $derived(wellKnowns.pickable.filter((wk) => matchesCard(wk, search)));

	const onDragStart = (event: DragEvent, nodeType: string) => {
		if (!event.dataTransfer) {
			return null;
		}

		type.current = nodeType;

		event.dataTransfer.effectAllowed = 'move';
	};
</script>

<div class="nodes-container items-start justify-start">
  {#if canAdd}
  <input
    type="search"
    bind:value={search}
    placeholder={m['nodes.search']()}
    aria-label={m['nodes.search']()}
    class="w-full rounded-md border border-white/40 bg-white/10 px-2 py-1.5 text-sm text-main-text placeholder:text-main-text/70 focus:border-white focus:outline-none"
    onkeydown={(event) => {
      if (event.key === 'Escape') search = '';
    }}
  />
  <!-- A card wired to any API, without a card type: always offered. -->
  <SidebarElement
    class="border-dashed! border-white/70! bg-transparent! font-semibold"
    ondragstart={(event) => onDragStart(event, ONE_OFF)}
  >
    {#snippet tooltipContent()}
      {m['one_off.sidebar_hint']()}
    {/snippet}
    <span class="mr-1 font-mono text-xs">{'{ }'}</span>
    {m['one_off.sidebar']()}
  </SidebarElement>
  {#each shown as wk (wk.key)}
    {@const description = wk[`description_${getLocale()}`]}
    <SidebarElement
      class={wk.key}
      ondragstart={(event) => onDragStart(event, wk.key)}
    >
      {#snippet tooltipContent()}
        {wk[`name_${getLocale()}`]}{#if description}: {description}{/if}
        {#if wk.scope}
          ({m[`postit_templates.scope_${wk.scope.kind}`]()})
        {/if}
      {/snippet}
      {#if wk.scope}
        <span class="mr-1 rounded bg-white/20 px-1 text-[10px] uppercase">{m[`postit_templates.badge_${wk.scope.kind}`]()}</span>
      {/if}
      {wk[`name_${getLocale()}`]}
    </SidebarElement>
  {:else}
    {#if search.trim()}
      <p class="px-1 text-sm text-main-text opacity-80" role="status">{m['nodes.search_empty']({ query: search.trim() })}</p>
    {/if}
  {/each}
  {:else}
    <PlanHint
      class="px-1 text-main-text"
      text={m['billing.hint.nodes']({ count: boardPlan.nodes, max: boardPlan.limits.nodes_per_board })}
    />
  {/if}

  <button class="sidebar-button gap-1" onclick={() => setManagingCardTypes(true)} tabindex="0">
    <Icon name="plus" class="w-5 h-5 text-main-text!" />
    {m['nodes.create']()}
  </button>
</div>
