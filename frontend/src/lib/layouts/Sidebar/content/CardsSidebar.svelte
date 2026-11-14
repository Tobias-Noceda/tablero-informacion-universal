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

	const type = useDnD();

	// The catalogue comes from the backend; the board's load usually has it already.
	wellKnowns.ensure().catch(() => {});

	// At the plan's card limit there is nothing to drop, only a line saying so.
	const canAdd = $derived(canAddNode(boardPlan.limits, boardPlan.nodes));

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
  <!-- A card wired to any API, without a card type. -->
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
  {#each wellKnowns.pickable as wk (wk.key)}
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
