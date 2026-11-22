// How long an edit takes to reach the other members of a board.
//
// For each number of clients N: signs in, makes a board with one note, opens
// N board sockets (as N browser tabs would), then moves the note MOVES times,
// one every INTERVAL_MS, and records when each socket hears the `update`.
// Latency is measured on this process's clock only:
//   from_send  request sent → update received (what a person sees)
//   from_ok    200 received → update received (the plan's criterion; it can
//              be negative: the change stream may beat the HTTP response)
// With SAME_CLOCK=1 (realtime sharing this machine's clock: Docker on Linux,
// or realtime run natively; not Docker Desktop, whose VM keeps its own clock,
// which the script detects) it also splits it with the emission time `ts`:
//   to_emit    request sent → realtime emits (backend + Mongo + change stream)
//   to_client  realtime emits → update received (socket.io)
// Then CONFLICTS rounds of two simultaneous moves of the same note check that
// the board ends on one of them and every client heard about it.
//
//   PERF_EMAIL=… PERF_PASSWORD=… node latency.mjs
//   CLIENTS=1,10,50,100 MOVES=200 INTERVAL_MS=1000 CONFLICTS=20 BASE_URL=http://localhost SAME_CLOCK=1
import { execFileSync } from 'node:child_process';
import os from 'node:os';

import { BASE_URL, boards, cards, openSocket, percentiles, save, signIn, sleep } from './lib.mjs';

const LEVELS = (process.env.CLIENTS ?? '1,10,50,100').split(',').map(Number);
const MOVES = Number(process.env.MOVES ?? 200);
const INTERVAL_MS = Number(process.env.INTERVAL_MS ?? 1000);
const CONFLICTS = Number(process.env.CONFLICTS ?? 20);
const SAME_CLOCK = process.env.SAME_CLOCK === '1';
const WAIT_MS = 5000; // an update later than this counts as lost
const OPEN_BATCH = 25;

// Latencies use the monotonic clock; the split against the server's `ts`
// (its Date.now()) needs the wall clock, since performance.timeOrigin +
// performance.now() drifts away from Date.now() as the system clock is adjusted.
const now = () => performance.now();

function commit() {
	try {
		return execFileSync('git', ['rev-parse', '--short', 'HEAD'], { encoding: 'utf-8' }).trim();
	} catch {
		return 'unknown';
	}
}

async function openAll(board, token, n, onUpdate) {
	const sockets = [];
	for (let i = 0; i < n; i += OPEN_BATCH) {
		const batch = Array.from({ length: Math.min(OPEN_BATCH, n - i) }, (_, k) =>
			openSocket(board, () => token, (ts) => onUpdate(i + k, ts))
		);
		sockets.push(...(await Promise.all(batch)));
	}
	return sockets;
}

// One round: a write and the N arrivals it causes.
function newRound(n) {
	let done;
	const round = {
		arrivals: new Array(n).fill(null),
		wall: new Array(n).fill(null),
		emitted: null,
		heard: 0,
		sent: 0,
		sentWall: 0,
		ok: 0,
		finished: new Promise((resolve) => (done = resolve)),
		hear(index, ts) {
			if (round.arrivals[index] !== null) return;
			round.arrivals[index] = now();
			round.wall[index] = Date.now();
			round.emitted ??= ts;
			if (++round.heard === n) done();
		}
	};
	return round;
}

async function measure(n) {
	const token = await signIn();
	const board = await boards.create(token, `perf latency ${n}`);
	const card = await cards.note(token, board.id, 'moving');
	let current = null;
	const sockets = await openAll(board.id, token, n, (i, ts) => current?.hear(i, ts));
	console.log(`N=${n}: ${sockets.length} sockets open`);

	const fromSend = [];
	const fromOk = [];
	const toEmit = [];
	const toClient = [];
	let lost = 0;

	try {
		for (let move = 0; move < MOVES; move++) {
			const round = newRound(n);
			current = round;
			round.sentWall = Date.now();
			round.sent = now();
			await cards.move(token, card.id, 100 + (move % 50) * 10, 100 + Math.floor(move / 50) * 10);
			round.ok = now();
			await Promise.race([round.finished, sleep(WAIT_MS)]);
			current = null;

			round.arrivals.forEach((arrival, socket) => {
				if (arrival === null) return void lost++;
				fromSend.push(arrival - round.sent);
				fromOk.push(arrival - round.ok);
				if (SAME_CLOCK && round.emitted) toClient.push(round.wall[socket] - round.emitted);
			});
			if (SAME_CLOCK && round.emitted) toEmit.push(round.emitted - round.sentWall);
			await sleep(Math.max(0, INTERVAL_MS - (now() - round.sent)));
		}

		const conflicts = await conflictRounds(token, board.id, card.id, sockets.length, (listener) => (current = listener));
		return {
			clients: n,
			moves: MOVES,
			lost,
			from_send_ms: percentiles(fromSend),
			from_ok_ms: percentiles(fromOk),
			...(SAME_CLOCK && {
				to_emit_ms: percentiles(toEmit),
				to_client_ms: percentiles(toClient),
				// A negative leg means the two clocks differ (Docker Desktop runs
				// containers in a VM with its own clock): discard the split.
				clock_skew: toClient.some((v) => v < 0) || toEmit.some((v) => v < 0)
			}),
			conflicts
		};
	} finally {
		for (const socket of sockets) socket.close();
		await boards.remove(token, board.id).catch((err) => console.warn('could not delete the board:', err.message));
	}
}

// Two members move the same note at once: the board must end on one of the
// two positions (the last write wins) and every client must hear a change.
async function conflictRounds(token, board, card, n, listen) {
	let diverged = 0;
	let unheard = 0;
	for (let i = 0; i < CONFLICTS; i++) {
		const first = { x: 1000 + i, y: 1000 };
		const second = { x: 2000 + i, y: 2000 };
		const round = newRound(n);
		listen(round);
		await Promise.all([cards.move(token, card, first.x, first.y), cards.move(token, card, second.x, second.y)]);
		await Promise.race([round.finished, sleep(WAIT_MS)]);
		listen(null);
		unheard += n - round.heard;
		const { position } = (await boards.get(token, board)).postits.find((entry) => entry.id === card);
		const landed = [first, second].some((spot) => spot.x === position.x && spot.y === position.y);
		if (!landed) diverged++;
		await sleep(200);
	}
	return { rounds: CONFLICTS, diverged, unheard };
}

const started = new Date().toISOString();
const results = [];
for (const n of LEVELS) {
	const result = await measure(n);
	results.push(result);
	const fmt = (s) => (s?.n ? `p50 ${s.p50}  p95 ${s.p95}  p99 ${s.p99}  max ${s.max}` : '-');
	console.log(`N=${n}  from_send: ${fmt(result.from_send_ms)}  |  from_ok: ${fmt(result.from_ok_ms)}  |  lost ${result.lost}`);
	if (SAME_CLOCK) {
		console.log(`        to_emit: ${fmt(result.to_emit_ms)}  |  to_client: ${fmt(result.to_client_ms)}`);
		if (result.clock_skew) console.warn('        the realtime clock differs from this one (a negative leg): ignore to_emit/to_client');
	}
	console.log(`        conflicts: ${result.conflicts.diverged} diverged, ${result.conflicts.unheard} unheard of ${result.conflicts.rounds}`);
}

const file = save('latency', {
	started,
	commit: commit(),
	host: { platform: os.platform(), cpus: os.cpus()[0]?.model, cores: os.cpus().length, memory_gb: Math.round(os.totalmem() / 2 ** 30) },
	settings: { base_url: BASE_URL, moves: MOVES, interval_ms: INTERVAL_MS, same_clock: SAME_CLOCK, wait_ms: WAIT_MS },
	results
});
console.log(`saved ${file}`);
process.exit(0);
