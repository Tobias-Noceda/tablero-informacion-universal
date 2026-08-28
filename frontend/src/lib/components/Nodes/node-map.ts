import AirQualityNode from "./AirQuality/AirQuality.svelte";
import CryptoPriceNode from "./CryptoPrice/CryptoPrice.svelte";
import ExchangeRateNode from "./ExchangeRate/ExchangeRate.svelte";
import DogFactsNode from "./DogFacts/DogFacts.svelte";
import DolarOficialNode from "./DolarOficial/DolarOficial.svelte";
import EventsSearchNode from "./EventsSearch/EventsSearch.svelte";
import GithubRepoNode from "./GithubRepo/GithubRepo.svelte";
import RiesgoPaisNode from "./RiesgoPais/RiesgoPais.svelte";
import StaticCardNode from "./StaticCard/StaticCard.svelte";
import TemperatureNode from "./Temperature/Temperature.svelte";

export const nodesMap = {
    "static_card": StaticCardNode,
    "temperature": TemperatureNode,
    "events_search": EventsSearchNode,
    "dog_facts": DogFactsNode,
    "dolar_oficial": DolarOficialNode,
    "riesgo_pais": RiesgoPaisNode,
    "crypto_price": CryptoPriceNode,
    "air_quality": AirQualityNode,
    "github_repo": GithubRepoNode,
    "exchange_rate": ExchangeRateNode
};

export type NodeParameter = {
    /** Exact key the backend expects in the well-known's Params map. */
    key: string;
    label: string;
    /** "secret" renders a picker over the board's stored credentials. */
    type: "string" | "number" | "secret";
    placeholder?: string;
    /** Prefilled value. A parameter without one must be supplied by the user. */
    default?: string;
};

export const parameters: Record<string, NodeParameter[]> = {
    "static_card": [
        { key: "text", label: "Text", type: "string", placeholder: "Enter text" }
    ],
    "temperature": [
        { key: "$latitude", label: "Latitude", type: "number", default: "-34.6131" },
        { key: "$longitude", label: "Longitude", type: "number", default: "-58.3772" },
        { key: "$start_date", label: "Start date", type: "string", placeholder: "YYYY-MM-DD" },
        { key: "$end_date", label: "End date", type: "string", placeholder: "YYYY-MM-DD" }
    ],
    "events_search": [
        { key: "$keyword", label: "Keyword", type: "string", placeholder: "Enter keyword" },
        { key: "$credential", label: "Credential", type: "secret" }
    ],
    "dog_facts": [],
    "dolar_oficial": [],
    "riesgo_pais": [],
    "crypto_price": [
        { key: "$coin", label: "Coin", type: "string", placeholder: "bitcoin", default: "bitcoin" },
        { key: "$currency", label: "Currency", type: "string", placeholder: "usd", default: "usd" }
    ],
    "air_quality": [
        { key: "$latitude", label: "Latitude", type: "number", default: "-34.6131" },
        { key: "$longitude", label: "Longitude", type: "number", default: "-58.3772" }
    ],
    "github_repo": [
        { key: "$query", label: "Repository", type: "string", placeholder: "itchyny/gojq" }
    ],
    "exchange_rate": [
        { key: "$base", label: "Base currency", type: "string", placeholder: "USD", default: "USD" },
        { key: "$currency", label: "Quote currency", type: "string", placeholder: "ARS", default: "ARS" },
        { key: "$credential", label: "Credential", type: "secret" }
    ]
};
