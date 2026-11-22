// How long a card takes to answer, with the cache cold and warm.
//
//   cold  COLD cards, each run once: the backend calls the source (which
//         answers after SOURCE_DELAY_MS) and caches the result
//   warm  WARM_VUS users read WARM cards already cached for WARM_SECONDS
//
// Criteria (the plan, §3.2 bis): warm p95 < 50 ms; cold p95 < source delay
// + 300 ms. At the end the source's own count says how many executions
// reached it: it must be one per cold card plus one warm-up per warm card,
// i.e. the warm phase never left the cache.
//
//   docker compose -f docker-compose.yaml -f perf/docker-compose.perf.yaml run --rm k6 run /scripts/cards.js
import exec from 'k6/execution';
import http from 'k6/http';
import { check } from 'k6';

import { SOURCE_DELAY_MS, createBoard, createCard, deleteBoard, executeParams, executeUrl, signIn, sourceHits } from './lib.js';

const COLD = Number(__ENV.COLD || 50);
const WARM = Number(__ENV.WARM || 10);
const WARM_VUS = Number(__ENV.WARM_VUS || 10);
const WARM_SECONDS = Number(__ENV.WARM_SECONDS || 60);
// The cold phase runs alone first; its budget is generous on purpose.
const COLD_BUDGET = Math.ceil((COLD * (SOURCE_DELAY_MS + 500)) / 1000) + 5;

export const options = {
	scenarios: {
		cold: { executor: 'shared-iterations', exec: 'cold', vus: 1, iterations: COLD, maxDuration: `${COLD_BUDGET}s` },
		warm: { executor: 'constant-vus', exec: 'warm', vus: WARM_VUS, duration: `${WARM_SECONDS}s`, startTime: `${COLD_BUDGET}s` }
	},
	thresholds: {
		'http_req_duration{cache:cold}': [`p(95)<${SOURCE_DELAY_MS + 300}`],
		'http_req_duration{cache:warm}': ['p(95)<50'],
		'http_req_failed{cache:cold}': ['rate==0'],
		'http_req_failed{cache:warm}': ['rate==0']
	},
	summaryTrendStats: ['avg', 'med', 'p(95)', 'p(99)', 'max']
};

export function setup() {
	const token = signIn();
	const board = createBoard(token, 'perf cards');
	const cold = Array.from({ length: COLD }, (_, i) => createCard(token, board, i));
	const warm = Array.from({ length: WARM }, (_, i) => createCard(token, board, COLD + i));
	for (const id of warm) http.get(executeUrl(id), executeParams(token, { kind: 'setup' }));
	return { token, board, cold, warm, hitsBefore: sourceHits() };
}

export function cold({ token, cold }) {
	const id = cold[exec.scenario.iterationInTest];
	const res = http.get(executeUrl(id), executeParams(token, { cache: 'cold' }));
	check(res, { 'cold card answers 200': (response) => response.status === 200 });
}

export function warm({ token, warm }) {
	const id = warm[Math.floor(Math.random() * warm.length)];
	const res = http.get(executeUrl(id), executeParams(token, { cache: 'warm' }));
	check(res, { 'warm card answers 200': (response) => response.status === 200 });
}

export function teardown({ token, board, cold, warm, hitsBefore }) {
	const hitsAfter = sourceHits();
	if (hitsAfter !== null) {
		// hitsBefore was read after the warm-up, so only the cold cards count.
		const reached = hitsAfter - hitsBefore;
		console.log(`the source served ${reached} requests for ${cold.length} cold cards (${warm.length} warm cards warmed up before)`);
		check(reached, { 'only cold executions reached the source': (n) => n === cold.length });
	}
	deleteBoard(token, board);
}
