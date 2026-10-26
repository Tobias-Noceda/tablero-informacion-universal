import AirQualityNode from './AirQuality/AirQuality.svelte';
import CryptoPriceNode from './CryptoPrice/CryptoPrice.svelte';
import ExchangeRateNode from './ExchangeRate/ExchangeRate.svelte';
import DogFactsNode from './DogFacts/DogFacts.svelte';
import DolarOficialNode from './DolarOficial/DolarOficial.svelte';
import EventsSearchNode from './EventsSearch/EventsSearch.svelte';
import GithubRepoNode from './GithubRepo/GithubRepo.svelte';
import GmailInboxNode from './GmailInbox/GmailInbox.svelte';
import GoogleCalendarNode from './GoogleCalendar/GoogleCalendar.svelte';
import NasaApodNode from './NasaApod/NasaApod.svelte';
import RiesgoPaisNode from './RiesgoPais/RiesgoPais.svelte';
import StaticCardNode from './StaticCard/StaticCard.svelte';
import TemperatureNode from './Temperature/Temperature.svelte';

export const nodesMap = {
	static_card: StaticCardNode,
	temperature: TemperatureNode,
	events_search: EventsSearchNode,
	dog_facts: DogFactsNode,
	dolar_oficial: DolarOficialNode,
	riesgo_pais: RiesgoPaisNode,
	crypto_price: CryptoPriceNode,
	air_quality: AirQualityNode,
	github_repo: GithubRepoNode,
	exchange_rate: ExchangeRateNode,
	gmail_inbox: GmailInboxNode,
	google_calendar: GoogleCalendarNode,
	nasa_apod: NasaApodNode
};

export type NodeParameter = {
	/** Exact key the backend expects in the well-known's Params map. */
	key: string;
	label_en: string;
	label_es: string;
	/** "secret" renders a picker over the board's stored credentials. */
	type: 'string' | 'number' | 'secret';
	placeholder_en?: string;
	placeholder_es?: string;
	/** Prefilled value. A parameter without one must be supplied by the user. */
	default?: string;
};

export const parameters: Record<string, NodeParameter[]> = {
	static_card: [
		{ key: 'text', label_en: 'Content', label_es: 'Contenido', type: 'string', placeholder_en: 'Enter text', placeholder_es: 'Ingrese texto', default: '' }],
	temperature: [
		{ key: '$latitude', label_en: 'Latitude', label_es: 'Latitud', type: 'number', default: '-34.6131' },
		{ key: '$longitude', label_en: 'Longitude', label_es: 'Longitud', type: 'number', default: '-58.3772' },
		{ key: '$start_date', label_en: 'Start date', label_es: 'Fecha de inicio', type: 'string', placeholder_en: 'YYYY-MM-DD', placeholder_es: 'AAAA-MM-DD' },
		{ key: '$end_date', label_en: 'End date', label_es: 'Fecha de fin', type: 'string', placeholder_en: 'YYYY-MM-DD', placeholder_es: 'AAAA-MM-DD' }
	],
	events_search: [
		{ key: '$keyword', label_en: 'Keyword', label_es: 'Palabra clave', type: 'string', placeholder_en: 'Enter keyword', placeholder_es: 'Ingrese palabra clave' },
		{ key: '$credential', label_en: 'Credential', label_es: 'Credencial', type: 'secret' }
	],
	dog_facts: [],
	dolar_oficial: [],
	riesgo_pais: [],
	crypto_price: [
		{ key: '$coin', label_en: 'Coin', label_es: 'Moneda', type: 'string', placeholder_en: 'bitcoin', placeholder_es: 'bitcoin', default: 'bitcoin' },
		{ key: '$currency', label_en: 'Currency', label_es: 'Moneda', type: 'string', placeholder_en: 'usd', placeholder_es: 'usd', default: 'usd' }
	],
	air_quality: [
		{ key: '$latitude', label_en: 'Latitude', label_es: 'Latitud', type: 'number', default: '-34.6131' },
		{ key: '$longitude', label_en: 'Longitude', label_es: 'Longitud', type: 'number', default: '-58.3772' }
	],
	github_repo: [
		{ key: '$query', label_en: 'Repository', label_es: 'Repositorio', type: 'string', placeholder_en: 'itchyny/gojq', placeholder_es: 'itchyny/gojq' }
	],
	exchange_rate: [
		{ key: '$base', label_en: 'Base currency', label_es: 'Moneda base', type: 'string', placeholder_en: 'USD', placeholder_es: 'USD', default: 'USD' },
		{
			key: '$currency',
			label_en: 'Quote currency',
			label_es: 'Moneda de cotización',
			type: 'string',
			placeholder_en: 'ARS',
			placeholder_es: 'ARS',
			default: 'ARS'
		},
		{ key: '$credential', label_en: 'Credential', label_es: 'Credencial', type: 'secret' }
	],
	gmail_inbox: [{ key: '$credential', label_en: 'Google account', label_es: 'Cuenta de Google', type: 'secret' }],
	google_calendar: [
		{ key: '$time_min', label_en: 'From', label_es: 'De', type: 'string', placeholder_en: '2026-01-01T00:00:00Z', placeholder_es: '2026-01-01T00:00:00Z' },
		{ key: '$credential', label_en: 'Google account', label_es: 'Cuenta de Google', type: 'secret' }
	],
	// The API key is the platform's own, injected server-side; nothing to ask for.
	nasa_apod: []
};

export type NodeOutput = {
	/** Key of the card's response, usable in its title as `{{key}}`. */
	key: string;
	label_en: string;
	label_es: string;
	/** Sample value, only to preview a title before the card exists. */
	example_en: unknown;
	example_es: unknown;
};

// Mirrors the Query keys of each template in
// backend/common/services/postits/well-knowns.go (a card without a resource,
// like static_card, answers with its params instead).
export const outputs: Record<string, NodeOutput[]> = {
	static_card: [{ key: 'text', label_en: 'Text', label_es: 'Texto', example_en: 'Hello!', example_es: 'Hola!' }],
	temperature: [
		{ key: 'min', label_en: 'Minimum °C', label_es: 'Mínimo °C', example_en: 12.4, example_es: 12.4 },
		{ key: 'max', label_en: 'Maximum °C', label_es: 'Máximo °C', example_en: 21.9, example_es: 21.9 }
	],
	events_search: [
		{ key: 'name', label_en: 'Event name', label_es: 'Nombre del evento', example_en: 'Coldplay - Music of the Spheres', example_es: 'Coldplay - Música de las Esferas' },
		{ key: 'sales', label_en: 'Sales start', label_es: 'Inicio de ventas', example_en: '2026-03-01T13:00:00Z', example_es: '2026-03-01T13:00:00Z' },
		{ key: 'image', label_en: 'Image URL', label_es: 'URL de la imagen', example_en: 'https://s1.ticketm.net/img.jpg', example_es: 'https://s1.ticketm.net/img.jpg' },
		{ key: 'date', label_en: 'Event date', label_es: 'Fecha del evento', example_en: '2026-08-01', example_es: '2026-08-01' },
		{ key: 'venue', label_en: 'Venue name', label_es: 'Nombre del lugar', example_en: 'Movistar Arena', example_es: 'Movistar Arena' },
		{ key: 'city', label_en: "Event's city", label_es: "Ciudad del evento", example_en: 'Buenos Aires', example_es: 'Buenos Aires' },
		{ key: 'url', label_en: "Event's url", label_es: "URL del evento", example_en: 'https://www.ticketmaster.com', example_es: 'https://www.ticketmaster.com' }
	],
	dog_facts: [{ key: 'body', label_en: 'Fact', label_es: 'Hecho', example_en: 'Dogs have three eyelids.', example_es: 'Los perros tienen tres párpados.' }],
	dolar_oficial: [
		{ key: 'compra', label_en: 'Buy', label_es: 'Comprar', example_en: 1420, example_es: 1420 },
		{ key: 'venta', label_en: 'Sell', label_es: 'Vender', example_en: 1470, example_es: 1470 }
	],
	riesgo_pais: [
		{ key: 'valor', label_en: 'Value', label_es: 'Valor', example_en: 612, example_es: 612 },
		{ key: 'fecha', label_en: 'Date', label_es: 'Fecha', example_en: '2026-10-03', example_es: '2026-10-03' }
	],
	crypto_price: [
		{ key: 'coin', label_en: 'Coin', label_es: 'Moneda', example_en: 'bitcoin', example_es: 'bitcoin' },
		{ key: 'price', label_en: 'Price', label_es: 'Precio', example_en: 62150, example_es: 62150 },
		{ key: 'change', label_en: '24h change', label_es: 'Cambio 24h', example_en: -1.23, example_es: -1.23 }
	],
	air_quality: [
		{ key: 'aqi', label_en: 'US AQI', label_es: 'AQI de EE.UU.', example_en: 42, example_es: 42 },
		{ key: 'pm25', label_en: 'PM2.5', label_es: 'PM2.5', example_en: 8.3, example_es: 8.3 },
		{ key: 'time', label_en: 'Measured at', label_es: 'Medido a las', example_en: '2026-10-04T12:00', example_es: '2026-10-04T12:00' }
	],
	github_repo: [
		{ key: 'name', label_en: 'Repository', label_es: 'Repositorio', example_en: 'itchyny/gojq', example_es: 'itchyny/gojq' },
		{ key: 'stars', label_en: 'Stars', label_es: 'Estrellas', example_en: 3412, example_es: 3412 },
		{ key: 'forks', label_en: 'Forks', label_es: 'Bifurcaciones', example_en: 245, example_es: 245 },
		{ key: 'issues', label_en: 'Open issues', label_es: 'Problemas abiertos', example_en: 12, example_es: 12 },
		{ key: 'description', label_en: 'Description', label_es: 'Descripción', example_en: 'Pure Go implementation of jq', example_es: 'Pure Go implementation of jq' }
	],
	exchange_rate: [
		{ key: 'code', label_en: 'Currency code', label_es: 'Código de moneda', example_en: 'ARS', example_es: 'ARS' },
		{ key: 'value', label_en: 'Rate', label_es: 'Tasa', example_en: 1452.5, example_es: 1452.5 },
		{ key: 'updated', label_en: 'Last update', label_es: 'Última actualización', example_en: '2026-10-04T00:00:00Z', example_es: '2026-10-04T00:00:00Z' }
	],
	gmail_inbox: [
		{ key: 'unread', label_en: 'Unread', label_es: 'Sin leer', example_en: 7, example_es: 7 },
		{ key: 'total', label_en: 'Total', label_es: 'Total', example_en: 1532, example_es: 1532 }
	],
	google_calendar: [
		{ key: 'summary', label_en: 'Next event', label_es: 'Próximo evento', example_en: 'Thesis meeting', example_es: 'Reunión de tesis' },
		{ key: 'start', label_en: 'Starts at', label_es: 'Comienza a las', example_en: '2026-10-05T15:00:00-03:00', example_es: '2026-10-05T15:00:00-03:00' }
	],
	nasa_apod: [
		{ key: 'title', label_en: 'Title', label_es: 'Título', example_en: 'The Horsehead Nebula', example_es: 'La Nebulosa del Caballo' },
		{ key: 'image', label_en: 'Image URL', label_es: 'URL de la imagen', example_en: 'https://apod.nasa.gov/apod/image/horsehead.jpg', example_es: 'https://apod.nasa.gov/apod/image/horsehead.jpg' },
		{ key: 'explanation', label_en: 'Explanation', label_es: 'Explicación', example_en: 'A dark cloud of dust...', example_es: 'Una nube oscura de polvo...' },
		{ key: 'date', label_en: 'Date', label_es: 'Fecha', example_en: '2026-10-04', example_es: '2026-10-04' }
	]
};
