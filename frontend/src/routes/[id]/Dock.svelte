<script lang="ts">
	import { m } from '$lib/paraglide/messages';
  import { useDnD } from './DnDProvider.svelte';

  const type = useDnD();

  const onDragStart = (event: DragEvent, nodeType: string) => {
    if (!event.dataTransfer) {
      return null;
    }

    type.current = nodeType;

    event.dataTransfer.effectAllowed = 'move';
  };
</script>

<aside>
  <div class="nodes-container items-center justify-center">
    <div
      class="input-node node"
      ondragstart={(event) => onDragStart(event, 'input')}
      draggable={true}
      role="button"
      tabindex="0"
    >
      {m["nodes.note"]()}
    </div>
    <div
      class="default-node node"
      ondragstart={(event) => onDragStart(event, 'default')}
      draggable={true}
      role="button"
      tabindex="0"
    >
      {m["nodes.smart_card"]()}
    </div>
  </div>
</aside>

<style>
  aside {
    width: 100%;
    font-size: 12px;
    display: flex;
    flex-direction: column;
    align-items: stretch;
  }

  .nodes-container {
    display: flex;
    flex-direction: row;
    gap: 1rem;
  }

  .node {
    margin: 0;
    padding: 0.5rem 1rem;
    font-weight: 700;
    border-radius: 5px;
    cursor: grab;
    background: var(--color-sidebar);
  }
</style>
