// Chapter 8 assumes 10,000 WebSockets per realtime task. This opens board
// sockets STEP at a time until one is refused, MAX are open, or the time to
// open a step goes over STEP_LIMIT_MS, and reports how far it got and how the
// handshake time grew. Run it only against local Docker: on Vercel it would
// bill function time and say little about a single task.
//
//   PERF_EMAIL=… PERF_PASSWORD=… node capacity.mjs
//   MAX=10000 STEP=250 STEP_LIMIT_MS=30000
//
// One process opens every socket: past a few thousand the machine running
// the script (ephemeral ports, file descriptors) can be the limit, not the
// service. The result says which side failed.
import { boards, openSocket, percentiles, save, signIn } from './lib.mjs';

const MAX = Number(process.env.MAX ?? 10_000);
const STEP = Number(process.env.STEP ?? 250);
const STEP_LIMIT_MS = Number(process.env.STEP_LIMIT_MS ?? 30_000);

const started = new Date().toISOString();
const token = await signIn();
const board = await boards.create(token, 'perf capacity');
const sockets = [];
const steps = [];
let stopped = `reached MAX=${MAX}`;

try {
	while (sockets.length < MAX) {
		const size = Math.min(STEP, MAX - sockets.length);
		const begin = performance.now();
		const times = [];
		const opened = await Promise.allSettled(
			Array.from({ length: size }, async () => {
				const t0 = performance.now();
				const socket = await openSocket(board.id, () => token, () => {});
				times.push(performance.now() - t0);
				return socket;
			})
		);
		const elapsed = performance.now() - begin;
		const refused = opened.filter((outcome) => outcome.status === 'rejected');
		sockets.push(...opened.filter((outcome) => outcome.status === 'fulfilled').map((outcome) => outcome.value));
		steps.push({ open: sockets.length, step_ms: Math.round(elapsed), handshake_ms: percentiles(times) });
		console.log(`${sockets.length} open, step ${Math.round(elapsed)} ms, handshake p95 ${percentiles(times).p95 ?? '-'} ms`);

		if (refused.length) {
			stopped = `${refused.length} refused at ${sockets.length}: ${refused[0].reason?.message ?? refused[0].reason}`;
			break;
		}
		if (elapsed > STEP_LIMIT_MS) {
			stopped = `a step took ${Math.round(elapsed)} ms (> ${STEP_LIMIT_MS})`;
			break;
		}
	}
} finally {
	for (const socket of sockets) socket.close();
	await boards.remove(token, board.id).catch(() => {});
}

console.log(`stopped: ${stopped}`);
console.log(`saved ${save('capacity', { started, settings: { max: MAX, step: STEP, step_limit_ms: STEP_LIMIT_MS }, open: sockets.length, stopped, steps })}`);
process.exit(0);
