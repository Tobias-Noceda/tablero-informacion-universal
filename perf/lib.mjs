// What the Node measurement scripts share: the API, the board socket as the
// frontend opens it, and the statistics.
import { randomUUID } from 'node:crypto';
import { mkdirSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { io } from 'socket.io-client';

export const BASE_URL = process.env.BASE_URL ?? 'http://localhost';
const RESULTS = path.join(path.dirname(fileURLToPath(import.meta.url)), 'results');

export function required(name) {
	const value = process.env[name];
	if (!value) throw new Error(`${name} is not set (see perf/README.md)`);
	return value;
}

export async function api(method, route, { body, token } = {}) {
	const res = await fetch(new URL('/api/v1' + route, BASE_URL), {
		method,
		headers: {
			'Content-Type': 'application/json',
			...(token && { Authorization: `Bearer ${token}` })
		},
		body: body === undefined ? undefined : JSON.stringify(body)
	});
	const text = await res.text();
	if (!res.ok) throw new Error(`${method} ${route}: ${res.status} ${text}`);
	return text ? JSON.parse(text) : undefined;
}

// A fresh access token for PERF_EMAIL (15 minutes; sign in again per run).
export async function signIn() {
	const session = await api('POST', '/auth/login', {
		body: { email: required('PERF_EMAIL'), password: required('PERF_PASSWORD') }
	});
	return session.access_token;
}

export const boards = {
	create: (token, name) => api('POST', '/boards', { body: { name }, token }),
	get: (token, id) => api('GET', `/boards/${id}`, { token }),
	remove: (token, id) => api('DELETE', `/boards/${id}`, { token })
};

export const cards = {
	note: (token, board, text) =>
		api('POST', '/post-its', {
			body: { board, well_known: 'static_card', title: { text, vars: false }, params: { text } },
			token
		}),
	move: (token, id, x, y) => api('PATCH', `/post-its/${id}/position`, { body: { x, y }, token })
};

// The board socket the way the frontend opens it ($modules/sockets.svelte.ts).
// forceNew: socket.io otherwise multiplexes every socket to the same origin
// over one connection, which would measure a single client N times.
// onUpdate receives the server's emission time (`ts`, the realtime clock).
export function openSocket(board, token, onUpdate) {
	const socket = io(new URL(BASE_URL).origin, {
		path: '/ws',
		transports: ['websocket'],
		query: { board, peer: randomUUID() },
		auth: (send) => send({ token: token() }),
		forceNew: true,
		reconnection: false,
		timeout: 20_000
	});
	socket.on('update', (update) => onUpdate(update.ts));
	socket.on('token_expiring', () => socket.emit('auth', { token: token() }));
	return new Promise((resolve, reject) => {
		socket.once('peers', () => resolve(socket));
		socket.once('connect_error', (err) => {
			socket.close();
			reject(err);
		});
	});
}

export function percentiles(values) {
	if (values.length === 0) return { n: 0 };
	const sorted = [...values].sort((a, b) => a - b);
	const at = (p) => sorted[Math.min(sorted.length - 1, Math.ceil((p / 100) * sorted.length) - 1)];
	const mean = sorted.reduce((sum, v) => sum + v, 0) / sorted.length;
	const round = (v) => Math.round(v * 10) / 10;
	return {
		n: sorted.length,
		mean: round(mean),
		p50: round(at(50)),
		p95: round(at(95)),
		p99: round(at(99)),
		max: round(sorted.at(-1))
	};
}

// Writes results/<name>-<timestamp>.json with what the report needs to say
// where the numbers come from (pass `started`, the commit and the hardware).
export function save(name, data) {
	mkdirSync(RESULTS, { recursive: true });
	const stamp = new Date().toISOString().replace(/[:.]/g, '-');
	const file = path.join(RESULTS, `${name}-${stamp}.json`);
	writeFileSync(
		file,
		JSON.stringify({ base_url: BASE_URL, node: process.version, ...data }, null, 2)
	);
	return file;
}

export const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
