// A controlled data source for the measurements: answers like a small JSON
// API after a fixed delay and counts what it served, so a run can tell how
// many card executions reached the source and how many the cache answered.
//
//   GET  /value?delay=<ms>   {"value": <n>, "served_at": <ms>} after <ms> (default 100)
//   GET  /stats              {"hits": <n>}
//   POST /stats/reset        sets hits back to 0
//
// No dependencies, so the container is just node:<version>-alpine.
import { createServer } from 'node:http';

const PORT = Number(process.env.PORT ?? 8080);
const MAX_DELAY_MS = 30_000;

let hits = 0;

function send(res, status, body) {
	res.writeHead(status, { 'Content-Type': 'application/json' });
	res.end(JSON.stringify(body));
}

const server = createServer((req, res) => {
	const url = new URL(req.url ?? '/', 'http://source');

	if (req.method === 'GET' && url.pathname === '/value') {
		hits++;
		const delay = Math.min(Math.max(Number(url.searchParams.get('delay') ?? 100) || 0, 0), MAX_DELAY_MS);
		setTimeout(() => send(res, 200, { value: hits, served_at: Date.now() }), delay);
		return;
	}
	if (req.method === 'GET' && url.pathname === '/stats') return send(res, 200, { hits });
	if (req.method === 'POST' && url.pathname === '/stats/reset') {
		hits = 0;
		return send(res, 200, { hits });
	}
	send(res, 404, { error: 'not_found' });
});

server.listen(PORT, () => console.log(`source listening on :${PORT}`));
process.on('SIGTERM', () => server.close(() => process.exit(0)));
