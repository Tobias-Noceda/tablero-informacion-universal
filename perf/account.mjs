// A verified Max account on the local Docker stack, for the measurements:
// registers through the API, reads the verification link the backend logs
// (start the stack without RESEND_API_KEY) and sets the plan in Mongo. Prints
// the variables to export. Never against production: there, use a real
// account on Max (PUT /system/users/:id/plan as an admin).
//
//   node account.mjs        (MONGO_DATABASE: the database the stack uses, default prod)
import { execFileSync } from 'node:child_process';
import { randomUUID } from 'node:crypto';
import { readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

import { BASE_URL, api, sleep } from './lib.mjs';

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const DATABASE = process.env.MONGO_DATABASE ?? 'prod';

if (!/^http:\/\/(localhost|127\.0\.0\.1)(:\d+)?$/.test(new URL(BASE_URL).origin)) {
	throw new Error(`account.mjs only works on the local Docker stack, not ${BASE_URL}`);
}

const compose = (...args) => execFileSync('docker', ['compose', ...args], { cwd: ROOT, encoding: 'utf-8' });
const env = (name) =>
	readFileSync(path.join(ROOT, '.env'), 'utf-8')
		.split(/\r?\n/)
		.find((line) => line.startsWith(`${name}=`))
		?.slice(name.length + 1)
		.trim() ?? '';

const email = `perf-${randomUUID().slice(0, 8)}@perf.test`;
const password = `perf-${randomUUID()}`;
await api('POST', '/auth/register', { body: { email, password, name: 'Perf', locale: 'en' } });

let token;
for (let attempt = 0; attempt < 30 && !token; attempt++) {
	const logs = compose('logs', 'backend', '--since', '5m', '--no-log-prefix');
	const at = logs.lastIndexOf(`mail to ${email}:`);
	token = at >= 0 && logs.slice(at).match(/verify\?token=([\w-]+)/)?.[1];
	if (!token) await sleep(300);
}
if (!token) throw new Error('no verification mail in the backend log (is RESEND_API_KEY empty?)');
await api('POST', '/auth/verify-email', { body: { token } });

const changed = compose(
	'exec', '-T', 'mongo', 'mongosh', '--quiet',
	'-u', env('MONGO_INITDB_ROOT_USERNAME'), '-p', env('MONGO_INITDB_ROOT_PASSWORD'),
	'--authenticationDatabase', 'admin',
	'--eval', `db.getSiblingDB('${DATABASE}').users.updateOne({email: '${email}'}, {$set: {plan: 'max'}}).modifiedCount`
).trim();
if (changed !== '1') throw new Error(`could not put ${email} on Max in database ${DATABASE}`);

console.log(`PERF_EMAIL=${email}`);
console.log(`PERF_PASSWORD=${password}`);
