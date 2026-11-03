import type { Component } from 'svelte';
import type { NodeProps } from '@xyflow/svelte';
import type { WellKnown } from '$types/api';

import AirQualityNode from './AirQuality/AirQuality.svelte';
import CryptoPriceNode from './CryptoPrice/CryptoPrice.svelte';
import ExchangeRateNode from './ExchangeRate/ExchangeRate.svelte';
import DogFactsNode from './DogFacts/DogFacts.svelte';
import DolarOficialNode from './DolarOficial/DolarOficial.svelte';
import EventsSearchNode from './EventsSearch/EventsSearch.svelte';
import GenericNode from './Generic/Generic.svelte';
import GithubRepoNode from './GithubRepo/GithubRepo.svelte';
import GmailInboxNode from './GmailInbox/GmailInbox.svelte';
import GoogleCalendarNode from './GoogleCalendar/GoogleCalendar.svelte';
import NasaApodNode from './NasaApod/NasaApod.svelte';
import RiesgoPaisNode from './RiesgoPais/RiesgoPais.svelte';
import StaticCardNode from './StaticCard/StaticCard.svelte';
import TemperatureNode from './Temperature/Temperature.svelte';

// Well-knowns with a hand-made card. What each one is (its request, form and
// outputs) comes from the backend: $stores/well-knowns.svelte.
export const nodesMap: Record<string, Component<NodeProps>> = {
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

// The canvas' node types: a hand-made card when there is one, the generic
// card for any other well-known. Cards already on the board keep their
// component even if their well-known was deleted since.
export function nodeTypesFor(list: WellKnown[], onBoard: (string | undefined)[] = []) {
	const types: Record<string, Component<NodeProps>> = { ...nodesMap };
	for (const key of [...list.map((wk) => wk.key), ...onBoard]) {
		if (key && !types[key]) types[key] = GenericNode;
	}
	return types;
}
