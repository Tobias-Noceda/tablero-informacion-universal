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
    <div
      class="riesgo_pais node"
      ondragstart={(event) => onDragStart(event, 'riesgo_pais')}
      draggable={true}
      role="button"
      tabindex="0"
    >
      {m["nodes.riesgo_pais"]()}
    </div>
    <div
      class="crypto_price node"
      ondragstart={(event) => onDragStart(event, 'crypto_price')}
      draggable={true}
      role="button"
      tabindex="0"
    >
      {m["nodes.crypto_price"]()}
    </div>
    <div
      class="air_quality node"
      ondragstart={(event) => onDragStart(event, 'air_quality')}
      draggable={true}
      role="button"
      tabindex="0"
    >
      {m["nodes.air_quality"]()}
    </div>
    <div
      class="exchange_rate node"
      ondragstart={(event) => onDragStart(event, 'exchange_rate')}
      draggable={true}
      role="button"
      tabindex="0"
    >
      {m["nodes.exchange_rate"]()}
    </div>
    <div
      class="github_repo node"
      ondragstart={(event) => onDragStart(event, 'github_repo')}
      draggable={true}
      role="button"
      tabindex="0"
    >
      {m["nodes.github_repo"]()}
    </div>
    <div
      class="gmail_inbox node"
      ondragstart={(event) => onDragStart(event, 'gmail_inbox')}
      draggable={true}
      role="button"
      tabindex="0"
    >
      {m["nodes.gmail_inbox"]()}
    </div>
    <div
      class="google_calendar node"
      ondragstart={(event) => onDragStart(event, 'google_calendar')}
      draggable={true}
      role="button"
      tabindex="0"
    >
      {m["nodes.google_calendar"]()}
    </div>
    <div
      class="nasa_apod node"
      ondragstart={(event) => onDragStart(event, 'nasa_apod')}
      draggable={true}
      role="button"
      tabindex="0"
    >
      {m["nodes.nasa_apod"]()}
    </div>
  </div>
</aside>

<style>
  aside {
    width: 100%;
    min-width: 0;
    font-size: 12px;
    display: flex;
    flex-direction: column;
    align-items: stretch;
    overflow-x: auto;
    overflow-y: hidden;
    z-index: 100;
    background: var(--color-background);
    padding: 10px;
  }

  .nodes-container {
    display: flex;
    flex-direction: row;
    gap: 1rem;
    /* "safe" keeps centering when it fits, but starts at the left edge when it overflows */
    justify-content: safe center;
    width: max-content;
    min-width: 100%;
  }

  .node {
    flex-shrink: 0;
    white-space: nowrap;
    margin: 0;
    padding: 0.5rem 1rem;
    font-weight: 700;
    border-radius: 5px;
    cursor: grab;
    background: var(--color-sidebar);
  }
</style>
