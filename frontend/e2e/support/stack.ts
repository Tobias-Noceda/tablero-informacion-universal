import { execFileSync } from 'node:child_process';
import { readFileSync } from 'node:fs';
import { randomUUID } from 'node:crypto';
import { fileURLToPath } from 'node:url';
import path from 'node:path';
import type { Page } from '@playwright/test';

import en from '../../messages/en.json' with { type: 'json' };

// What the e2e tests need from the stack beyond the browser: accounts made
// through the real API, the verification link the backend logs instead of
// mailing it (no RESEND_API_KEY), and a plan set in the database.

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../..');
export const BASE_URL = process.env.E2E_BASE_URL ?? 'http://localhost';
// The database the stack under test uses (MONGO_DATABASE when it was started).
const DATABASE = process.env.E2E_MONGO_DATABASE ?? 'e2e_ui';
export const PASSWORD = 'e2e password 123';

// A message of the English UI by its dotted key, so a copy change does not
// break the tests.
export function t(key: string, vars: Record<string, string | number> = {}): string {
	let node: unknown = en;
	for (const part of key.split('.')) node = (node as Record<string, unknown>)[part];
	if (typeof node !== 'string') throw new Error(`no message ${key}`);
	return node.replace(/\{(\w+)\}/g, (_, name) => String(vars[name] ?? `{${name}}`));
}

function compose(...args: string[]): string {
	return execFileSync('docker', ['compose', ...args], { cwd: ROOT, encoding: 'utf-8' });
}

function env(name: string): string {
	const line = readFileSync(path.join(ROOT, '.env'), 'utf-8')
		.split(/\r?\n/)
		.find((l) => l.startsWith(`${name}=`));
	return line ? line.slice(name.length + 1).trim() : '';
}

// Sign-up and sign-in are limited per client address. Behind the Docker ALB
// the backend trusts the private network's X-Forwarded-For (TRUSTED_PROXIES),
// so each account the tests make comes from an address of its own.
export function clientAddress(): string {
	const bytes = randomUUID().replace(/-/g, '');
	return `198.51.${parseInt(bytes.slice(0, 2), 16)}.${parseInt(bytes.slice(2, 4), 16)}`;
}

export type Account = { name: string; email: string; id: string; token: string; address: string };

async function api(
	method: string,
	route: string,
	body?: unknown,
	token?: string,
	address?: string
) {
	const res = await fetch(new URL('/api/v1' + route, BASE_URL), {
		method,
		headers: {
			'Content-Type': 'application/json',
			...(token && { Authorization: `Bearer ${token}` }),
			...(address && { 'X-Forwarded-For': address })
		},
		body: body === undefined ? undefined : JSON.stringify(body)
	});
	if (!res.ok) throw new Error(`${method} ${route}: ${res.status} ${await res.text()}`);
	const text = await res.text();
	return text ? JSON.parse(text) : undefined;
}

// The verification link the backend logged for email.
export async function verificationToken(email: string): Promise<string> {
	for (let attempt = 0; attempt < 30; attempt++) {
		const logs = compose('logs', 'backend', '--since', '5m', '--no-log-prefix');
		const at = logs.lastIndexOf(`mail to ${email}:`);
		const token = at >= 0 && logs.slice(at).match(/verify\?token=([\w-]+)/)?.[1];
		if (token) return token;
		await new Promise((resolve) => setTimeout(resolve, 300));
	}
	throw new Error(`no verification mail for ${email} in the backend log`);
}

// A new, verified account, on the plan given (free by default).
export async function account(
	name: string,
	plan: 'free' | 'pro' | 'max' = 'free'
): Promise<Account> {
	const email = `e2e-${name}-${randomUUID().slice(0, 8)}@it.test`;
	const address = clientAddress();
	await api(
		'POST',
		'/auth/register',
		{ email, password: PASSWORD, name, locale: 'en' },
		undefined,
		address
	);
	const session = await api('POST', '/auth/verify-email', {
		token: await verificationToken(email)
	});
	if (plan !== 'free') setPlan(email, plan);
	return { name, email, id: session.user.id, token: session.access_token, address };
}

export function setPlan(email: string, plan: 'free' | 'pro' | 'max') {
	const update = `db.getSiblingDB('${DATABASE}').users.updateOne({email: '${email}'}, {$set: {plan: '${plan}'}}).modifiedCount`;
	const changed = compose(
		'exec',
		'-T',
		'mongo',
		'mongosh',
		'--quiet',
		'-u',
		env('MONGO_INITDB_ROOT_USERNAME'),
		'-p',
		env('MONGO_INITDB_ROOT_PASSWORD'),
		'--authenticationDatabase',
		'admin',
		'--eval',
		update
	).trim();
	if (changed !== '1') throw new Error(`could not put ${email} on ${plan} (${changed})`);
}

// Signs in from the page's own context, so the refresh cookie lands in that
// browser as it would after the form.
export async function signIn(page: Page, who: Account) {
	const res = await page.request.post('/api/v1/auth/login', {
		data: { email: who.email, password: PASSWORD },
		headers: { 'X-Forwarded-For': who.address }
	});
	if (!res.ok()) throw new Error(`sign in ${who.email}: ${res.status()} ${await res.text()}`);
}

export const boards = {
	create: (who: Account, name: string) => api('POST', '/boards', { name }, who.token),
	get: (who: Account, id: string) => api('GET', `/boards/${id}`, undefined, who.token),
	share: (who: Account, id: string, email: string, role: 'editor' | 'viewer') =>
		api('PUT', `/boards/${id}/members`, { email, role }, who.token)
};

export const cards = {
	note: (who: Account, board: string, text: string) =>
		api(
			'POST',
			'/post-its',
			{ board, well_known: 'static_card', title: { text, vars: false }, params: { text } },
			who.token
		),
	custom: (who: Account, board: string, title: string, resource: string) =>
		api(
			'POST',
			'/post-its',
			{
				board,
				title: { text: title, vars: title.includes('{{') },
				resource,
				request: { method: 'GET' },
				response: 'json',
				query: { value: '.value' },
				rate: 60
			},
			who.token
		),
	move: (who: Account, id: string, x: number, y: number) =>
		api('PATCH', `/post-its/${id}/position`, { x, y }, who.token)
};
