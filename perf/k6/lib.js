// What the k6 scenarios share: the API and the cards they measure.
import http from 'k6/http';

export const BASE_URL = __ENV.BASE_URL || 'http://localhost';
// The fake source (perf/source/server.mjs). Against production set
// CARD_KIND=note instead: the backend there cannot reach it.
export const SOURCE_URL = __ENV.SOURCE_URL || 'http://203.0.113.10:8080';
export const CARD_KIND = __ENV.CARD_KIND || 'source';
export const SOURCE_DELAY_MS = Number(__ENV.SOURCE_DELAY_MS || 100);

function headers(token) {
	return { 'Content-Type': 'application/json', ...(token && { Authorization: `Bearer ${token}` }) };
}

export function api(method, route, body, token, tags = { kind: 'setup' }) {
	const res = http.request(method, `${BASE_URL}/api/v1${route}`, body === undefined ? null : JSON.stringify(body), {
		headers: headers(token),
		tags
	});
	if (res.status < 200 || res.status >= 300) throw new Error(`${method} ${route}: ${res.status} ${res.body}`);
	return res.body ? res.json() : null;
}

// An access token for PERF_EMAIL: valid 15 minutes, so keep a run shorter.
export function signIn() {
	if (!__ENV.PERF_EMAIL || !__ENV.PERF_PASSWORD) throw new Error('PERF_EMAIL and PERF_PASSWORD are not set (see perf/README.md)');
	return api('POST', '/auth/login', { email: __ENV.PERF_EMAIL, password: __ENV.PERF_PASSWORD }).access_token;
}

export function createBoard(token, name) {
	return api('POST', '/boards', { name }, token).id;
}

export function deleteBoard(token, id) {
	api('DELETE', `/boards/${id}`, undefined, token);
}

// A card that reads the fake source (rate 60, the Max plan's minimum, so its
// result stays cached a minute), or a note that makes no request.
export function createCard(token, board, index) {
	if (CARD_KIND === 'note') {
		const text = `note ${index}`;
		return api('POST', '/post-its', { board, well_known: 'static_card', title: { text, vars: false }, params: { text } }, token).id;
	}
	return api(
		'POST',
		'/post-its',
		{
			board,
			title: { text: 'Value: {{value}}', vars: true },
			resource: `${SOURCE_URL}/value?delay=${SOURCE_DELAY_MS}&card=${index}`,
			request: { method: 'GET' },
			response: 'json',
			query: { value: '.value' },
			rate: 60
		},
		token
	).id;
}

// What the board does for each card it shows: GET /post-its/:id runs it
// (or answers from the cache).
export function executeParams(token, tags) {
	return { headers: headers(token), tags };
}

export function executeUrl(id) {
	return `${BASE_URL}/api/v1/post-its/${id}`;
}

export function sourceHits() {
	if (CARD_KIND === 'note') return null;
	const res = http.get(`${SOURCE_URL}/stats`, { tags: { kind: 'setup' } });
	return res.status === 200 ? res.json().hits : null;
}
