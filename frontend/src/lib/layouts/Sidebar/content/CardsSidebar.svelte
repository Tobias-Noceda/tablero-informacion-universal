<script lang="ts">
  import '../index.css';

	import Icon from '$components/Icon/Icon.svelte';
	import { m } from '$lib/paraglide/messages';
	import { getLocale } from '$lib/paraglide/runtime';
	import { useDnD } from '$providers/DnDProvider.svelte';
	import { setManagingCardTypes } from '$stores/sidebar';
	import { wellKnowns } from '$stores/well-knowns.svelte';
	import SidebarElement from './SidebarElement.svelte';

	const type = useDnD();

	// The catalogue comes from the backend; the board's load usually has it already.
	wellKnowns.ensure().catch(() => {});

	const onDragStart = (event: DragEvent, nodeType: string) => {
		if (!event.dataTransfer) {
			return null;
		}

		type.current = nodeType;

		event.dataTransfer.effectAllowed = 'move';
	};
</script>

<div class="nodes-container items-start justify-start">
  {#each wellKnowns.pickable as wk (wk.key)}
    {@const description = wk[`description_${getLocale()}`]}
    <SidebarElement
      class={wk.key}
      ondragstart={(event) => onDragStart(event, wk.key)}
    >
      {#snippet tooltipContent()}
        {wk[`name_${getLocale()}`]}{#if description}: {description}{/if}
        {#if wk.scope}
          ({m[`custom_wk.scope_${wk.scope.kind}`]()})
        {/if}
      {/snippet}
      {#if wk.scope}
        <span class="mr-1 rounded bg-white/20 px-1 text-[10px] uppercase">{m[`custom_wk.badge_${wk.scope.kind}`]()}</span>
      {/if}
      {wk[`name_${getLocale()}`]}
    </SidebarElement>
  {/each}

  <button class="sidebar-button gap-1" onclick={() => setManagingCardTypes(true)} tabindex="0">
    <Icon name="plus" class="w-5 h-5 text-main-text!" />
    {m['nodes.create']()}
  </button>
</div>
