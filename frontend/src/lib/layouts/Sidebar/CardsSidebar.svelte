<script lang="ts">
  import './index.css';

	import Icon from '$components/Icon/Icon.svelte';
	import { m } from '$lib/paraglide/messages';
	import { useDnD } from '$providers/DnDProvider.svelte';
	import SidebarElement from './SidebarElement.svelte';

	const type = useDnD();

	const onDragStart = (event: DragEvent, nodeType: string) => {
		if (!event.dataTransfer) {
			return null;
		}

		type.current = nodeType;

		event.dataTransfer.effectAllowed = 'move';
	};
</script>

<div class="nodes-container items-start justify-start">
  <SidebarElement
    class="static_card"
    ondragstart={(event) => onDragStart(event, 'static_card')}
  >
    {m['nodes.static_card']()}
  </SidebarElement>
  <!-- <div
    class="temperature node"
    ondragstart={(event) => onDragStart(event, 'temperature')}
    draggable={true}
    role="button"
    tabindex="0"
  >
    {m["nodes.temperature"]()}
  </div> -->
  <SidebarElement
    class="events_search"
    ondragstart={(event) => onDragStart(event, 'events_search')}
  >
    {m['nodes.events_search']()}
  </SidebarElement>
  <SidebarElement
    class="dog_facts"
    ondragstart={(event) => onDragStart(event, 'dog_facts')}
  >
    {m['nodes.dog_facts']()}
  </SidebarElement>
  <SidebarElement
    class="dolar_oficial"
    ondragstart={(event) => onDragStart(event, 'dolar_oficial')}
  >
    {m['nodes.dolar_oficial']()}
  </SidebarElement>
  <SidebarElement
    class="riesgo_pais"
    ondragstart={(event) => onDragStart(event, 'riesgo_pais')}
  >
    {m['nodes.riesgo_pais']()}
  </SidebarElement>
  <SidebarElement
    class="crypto_price"
    ondragstart={(event) => onDragStart(event, 'crypto_price')}
  >
    {m['nodes.crypto_price']()}
  </SidebarElement>
  <SidebarElement
    class="air_quality"
    ondragstart={(event) => onDragStart(event, 'air_quality')}
  >
    {m['nodes.air_quality']()}
  </SidebarElement>
  <SidebarElement
    class="exchange_rate"
    ondragstart={(event) => onDragStart(event, 'exchange_rate')}
  >
    {m['nodes.exchange_rate']()}
  </SidebarElement>
  <SidebarElement
    class="github_repo"
    ondragstart={(event) => onDragStart(event, 'github_repo')}
  >
    {m['nodes.github_repo']()}
  </SidebarElement>
  <SidebarElement
    class="gmail_inbox"
    ondragstart={(event) => onDragStart(event, 'gmail_inbox')}
  >
    {m['nodes.gmail_inbox']()}
  </SidebarElement>
  <SidebarElement
    class="google_calendar"
    ondragstart={(event) => onDragStart(event, 'google_calendar')}
  >
    {m['nodes.google_calendar']()}
  </SidebarElement>
  <SidebarElement
    class="nasa_apod"
    ondragstart={(event) => onDragStart(event, 'nasa_apod')}
  >
    {m['nodes.nasa_apod']()}
  </SidebarElement>

  <button class="sidebar-button gap-1" onclick={() => console.log('Add new node')} tabindex="0">
    <Icon name="plus" class="w-5 h-5 text-main-text!" />
    {m['nodes.create']()}
  </button>
</div>
