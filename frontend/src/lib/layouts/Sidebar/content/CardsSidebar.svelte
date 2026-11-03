<script lang="ts">
  import '../index.css';

	import Icon from '$components/Icon/Icon.svelte';
	import { m } from '$lib/paraglide/messages';
	import { getLocale } from '$lib/paraglide/runtime';
	import { useDnD } from '$providers/DnDProvider.svelte';
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
  {#each wellKnowns.list as wk (wk.key)}
    {@const description = wk[`description_${getLocale()}`]}
    <SidebarElement
      class={wk.key}
      ondragstart={(event) => onDragStart(event, wk.key)}
    >
      {#snippet tooltipContent()}
        {wk[`name_${getLocale()}`]}{#if description}: {description}{/if}
      {/snippet}
      {wk[`name_${getLocale()}`]}
    </SidebarElement>
  {/each}

  <button class="sidebar-button gap-1" onclick={() => console.log('Add new node')} tabindex="0">
    <Icon name="plus" class="w-5 h-5 text-main-text!" />
    {m['nodes.create']()}
  </button>
</div>
