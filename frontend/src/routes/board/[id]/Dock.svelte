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
      class="static_card node"
      ondragstart={(event) => onDragStart(event, 'static_card')}
      draggable={true}
      role="button"
      tabindex="0"
    >
      {m["nodes.static_card"]()}
    </div>
    <!-- <div
      class="temperature node"
      ondragstart={(event) => onDragStart(event, 'temperature')}
      draggable={true}
      role="button"
      tabindex="0"
    >
      {m["nodes.temperature"]()}
    </div> -->
    <div
      class="events_search node"
      ondragstart={(event) => onDragStart(event, 'events_search')}
      draggable={true}
      role="button"
      tabindex="0"
    >
      {m["nodes.events_search"]()}
    </div>
    <div
      class="dog_facts node"
      ondragstart={(event) => onDragStart(event, 'dog_facts')}
      draggable={true}
      role="button"
      tabindex="0"
    >
      {m["nodes.dog_facts"]()}
    </div>
    <div
      class="dolar_oficial node"
      ondragstart={(event) => onDragStart(event, 'dolar_oficial')}
      draggable={true}
      role="button"
      tabindex="0"
    >
      {m["nodes.dolar_oficial"]()}
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
