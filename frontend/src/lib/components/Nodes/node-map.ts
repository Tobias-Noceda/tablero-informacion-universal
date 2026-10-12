import AirQualityNode from "./AirQuality/AirQuality.svelte";
import CryptoPriceNode from "./CryptoPrice/CryptoPrice.svelte";
import ExchangeRateNode from "./ExchangeRate/ExchangeRate.svelte";
import DogFactsNode from "./DogFacts/DogFacts.svelte";
import DolarOficialNode from "./DolarOficial/DolarOficial.svelte";
import EventsSearchNode from "./EventsSearch/EventsSearch.svelte";
import GithubRepoNode from "./GithubRepo/GithubRepo.svelte";
import GmailInboxNode from "./GmailInbox/GmailInbox.svelte";
import GoogleCalendarNode from "./GoogleCalendar/GoogleCalendar.svelte";
import NasaApodNode from "./NasaApod/NasaApod.svelte";
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
    "exchange_rate": ExchangeRateNode,
    "gmail_inbox": GmailInboxNode,
    "google_calendar": GoogleCalendarNode,
    "nasa_apod": NasaApodNode
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
    ],
    "gmail_inbox": [
        { key: "$credential", label: "Google account", type: "secret" }
    ],
    "google_calendar": [
        { key: "$time_min", label: "From", type: "string", placeholder: "2026-01-01T00:00:00Z" },
        { key: "$credential", label: "Google account", type: "secret" }
    ],
    // The API key is the platform's own, injected server-side; nothing to ask for.
    "nasa_apod": []
};

export type NodeOutput = {
    /** Key of the card's response, usable in its title as `{{key}}`. */
    key: string;
    label: string;
    /** Sample value, only to preview a title before the card exists. */
    example: unknown;
};

// Mirrors the Query keys of each template in
// backend/common/services/postits/well-knowns.go (a card without a resource,
// like static_card, answers with its params instead).
export const outputs: Record<string, NodeOutput[]> = {
    "static_card": [
        { key: "text", label: "Text", example: "Hello!" }
    ],
    "temperature": [
        { key: "min", label: "Minimum °C", example: 12.4 },
        { key: "max", label: "Maximum °C", example: 21.9 }
    ],
    "events_search": [
        { key: "name", label: "Event name", example: "Coldplay - Music of the Spheres" },
        { key: "sales", label: "Sales start", example: "2026-03-01T13:00:00Z" },
        { key: "image", label: "Image URL", example: "https://s1.ticketm.net/img.jpg" }
    ],
    "dog_facts": [
        { key: "body", label: "Fact", example: "Dogs have three eyelids." }
    ],
    "dolar_oficial": [
        { key: "compra", label: "Buy", example: 1420 },
        { key: "venta", label: "Sell", example: 1470 }
    ],
    "riesgo_pais": [
        { key: "valor", label: "Value", example: 612 },
        { key: "fecha", label: "Date", example: "2026-10-03" }
    ],
    "crypto_price": [
        { key: "coin", label: "Coin", example: "bitcoin" },
        { key: "price", label: "Price", example: 62150 },
        { key: "change", label: "24h change", example: -1.23 }
    ],
    "air_quality": [
        { key: "aqi", label: "US AQI", example: 42 },
        { key: "pm25", label: "PM2.5", example: 8.3 },
        { key: "time", label: "Measured at", example: "2026-10-04T12:00" }
    ],
    "github_repo": [
        { key: "name", label: "Repository", example: "itchyny/gojq" },
        { key: "stars", label: "Stars", example: 3412 },
        { key: "forks", label: "Forks", example: 245 },
        { key: "issues", label: "Open issues", example: 12 },
        { key: "description", label: "Description", example: "Pure Go implementation of jq" }
    ],
    "exchange_rate": [
        { key: "code", label: "Currency code", example: "ARS" },
        { key: "value", label: "Rate", example: 1452.5 },
        { key: "updated", label: "Last update", example: "2026-10-04T00:00:00Z" }
    ],
    "gmail_inbox": [
        { key: "unread", label: "Unread", example: 7 },
        { key: "total", label: "Total", example: 1532 }
    ],
    "google_calendar": [
        { key: "summary", label: "Next event", example: "Thesis meeting" },
        { key: "start", label: "Starts at", example: "2026-10-05T15:00:00-03:00" }
    ],
    "nasa_apod": [
        { key: "title", label: "Title", example: "The Horsehead Nebula" },
        { key: "image", label: "Image URL", example: "https://apod.nasa.gov/apod/image/horsehead.jpg" },
        { key: "explanation", label: "Explanation", example: "A dark cloud of dust..." },
        { key: "date", label: "Date", example: "2026-10-04" }
    ]
};
