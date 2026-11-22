// The API under a growing number of people, each doing
// what an open board does: GET /boards/:id, then run its CARDS cards at once,
// then wait THINK_SECONDS. Users step through STAGES (default 10, 50, 100,
// 200), STAGE_SECONDS each (default 120); every request is tagged with its
// stage, so the summary gives p95 and errors per step and shows where p95
// takes off.
//
// The cards read the fake source and stay cached a minute (rate 60), as on a
// real board: the run loads the backend, Redis and Mongo, and the source only
// once per card per minute. Keep the run under 15 minutes (one access token).
//
//   docker compose -f docker-compose.yaml -f perf/docker-compose.perf.yaml run --rm k6 run /scripts/load.js
//   against production: k6 run -e BASE_URL=https://… -e CARD_KIND=note -e STAGES=10,25 -e STAGE_SECONDS=300 perf/k6/load.js
import http from 'k6/http';
import { check, sleep } from 'k6';

import { BASE_URL, createBoard, createCard, deleteBoard, executeParams, executeUrl, signIn } from './lib.js';

const STAGES = (__ENV.STAGES || '10,50,100,200').split(',').map(Number);
const STAGE_SECONDS = Number(__ENV.STAGE_SECONDS || 120);
const RAMP_SECONDS = 10;
const CARDS = Number(__ENV.CARDS || 10);
const THINK_SECONDS = Number(__ENV.THINK_SECONDS || 10);

if (STAGES.length * STAGE_SECONDS > 14 * 60) throw new Error('keep the run under 15 minutes: the access token expires');

const thresholds = {};
STAGES.forEach((users) => {
	// A loose 5 s bound: these thresholds mainly make k6 report every stage.
	thresholds[`http_req_duration{stage:${users}}`] = ['p(95)<5000'];
	thresholds[`http_req_failed{stage:${users}}`] = ['rate<0.01'];
});

export const options = {
	scenarios: {
		load: {
			executor: 'ramping-vus',
			startVUs: 0,
			stages: STAGES.flatMap((users) => [
				{ target: users, duration: `${RAMP_SECONDS}s` },
				{ target: users, duration: `${STAGE_SECONDS - RAMP_SECONDS}s` }
			]),
			gracefulRampDown: '10s'
		}
	},
	thresholds,
	summaryTrendStats: ['avg', 'med', 'p(95)', 'p(99)', 'max']
};

export function setup() {
	const token = signIn();
	const board = createBoard(token, 'perf load');
	const cards = Array.from({ length: CARDS }, (_, i) => createCard(token, board, i));
	return { token, board, cards, start: Date.now() };
}

// The stage the run is in, by the time since the scenario started.
function stageAt(start) {
	const index = Math.min(STAGES.length - 1, Math.floor((Date.now() - start) / 1000 / STAGE_SECONDS));
	return String(STAGES[index]);
}

export default function ({ token, board, cards, start }) {
	const stage = stageAt(start);
	const opened = http.get(`${BASE_URL}/api/v1/boards/${board}`, executeParams(token, { stage, kind: 'board' }));
	check(opened, { 'board opens': (response) => response.status === 200 });

	const answers = http.batch(cards.map((id) => ['GET', executeUrl(id), null, executeParams(token, { stage, kind: 'card' })]));
	check(answers, { 'every card answers': (responses) => responses.every((response) => response.status === 200) });

	sleep(THINK_SECONDS);
}

export function teardown({ token, board }) {
	deleteBoard(token, board);
}
