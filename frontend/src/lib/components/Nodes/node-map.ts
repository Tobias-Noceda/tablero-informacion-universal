import DogFactsNode from "./DogFacts/DogFacts.svelte";
import DolarOficialNode from "./DolarOficial/DolarOficial.svelte";
import EventsSearchNode from "./EventsSearch/EventsSearch.svelte";
import StaticCardNode from "./StaticCard/StaticCard.svelte";
import TemperatureNode from "./Temperature/Temperature.svelte";

export const nodesMap = {
    "static_card": StaticCardNode,
    "temperature": TemperatureNode,
    "events_search": EventsSearchNode,
    "dog_facts": DogFactsNode,
    "dolar_oficial": DolarOficialNode
};

export const parameters = {
    "static_card": [
        { name: "text", type: "string" },
    ],
    "temperature": [
        { name: "latitude", type: "number" },
        { name: "longitude", type: "number" },
        { name: "start_date", type: "string" },
        { name: "end_date", type: "string" },
    ],
    "events_search": [
        { name: "keyword", type: "string" },
    ],
    "dog_facts": [],
    "dolar_oficial": []
};
