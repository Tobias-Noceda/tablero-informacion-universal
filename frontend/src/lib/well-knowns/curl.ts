// A cURL command, as API docs show their examples, read into a one-off
// card's request. Only what a card can send is kept (GET or POST, headers,
// query string, a JSON body); a credential found in it is never kept as
// plain text: its row becomes a credential to pick instead.

import type { OneOffMethod } from '$types/api';
import type { OneOffDraft } from './one-off';

export type CurlRow = { key: string; value: string; credential: boolean };

export type CurlRequest = {
	resource: string;
	method: OneOffMethod;
	queries: CurlRow[];
	headers: CurlRow[];
	body: string;
	/** Where a credential was found ("Authorization", "api_key", …): left for the user to pick. */
	credentials: string[];
	/** What the card cannot do and was left out. */
	dropped: ('method' | 'body_not_json' | 'flags')[];
};

export class CurlError extends Error {
	constructor(readonly code: 'not_curl' | 'no_url' | 'unclosed_quote' | 'bad_url') {
		super(code);
		this.name = 'CurlError';
	}
}

/** Splits a command line the way a POSIX shell would, plus `^` continuations from cmd.exe. */
export function tokenize(command: string): string[] {
	const text = command.replace(/\\\r?\n/g, ' ').replace(/\^\r?\n/g, ' ');
	const tokens: string[] = [];
	let current = '';
	let started = false;
	let quote: "'" | '"' | null = null;

	for (let i = 0; i < text.length; i++) {
		const char = text[i];
		if (quote === "'") {
			if (char === "'") quote = null;
			else current += char;
		} else if (quote === '"') {
			if (char === '"') quote = null;
			else if (char === '\\' && i + 1 < text.length && '"\\$`'.includes(text[i + 1]))
				current += text[++i];
			else current += char;
		} else if (char === "'" || char === '"') {
			quote = char;
			started = true;
		} else if (char === '\\' && i + 1 < text.length) {
			current += text[++i];
			started = true;
		} else if (/\s/.test(char)) {
			if (started) tokens.push(current);
			current = '';
			started = false;
		} else {
			current += char;
			started = true;
		}
	}
	if (quote) throw new CurlError('unclosed_quote');
	if (started) tokens.push(current);
	return tokens;
}

// Headers and query parameters whose value is a credential.
const SECRET_HEADER =
	/^(authorization|proxy-authorization|cookie|x-api-key|api-key|apikey|x-auth-token|x-access-token|.*-token|.*-key|.*secret.*)$/i;
const SECRET_QUERY =
	/^(api[_-]?key|apikey|key|token|access[_-]?token|auth|secret|client[_-]?secret|password|sig|signature)$/i;

// Flags whose value is not part of the request a card sends.
const SKIP_WITH_VALUE = new Set([
	'-o',
	'--output',
	'-m',
	'--max-time',
	'--connect-timeout',
	'-w',
	'--write-out',
	'-e',
	'--referer',
	'--retry',
	'-x',
	'--proxy',
	'--cacert',
	'--cert',
	'-E',
	'--key',
	'-c',
	'--cookie-jar',
	'-T',
	'--upload-file',
	'-F',
	'--form',
	'--limit-rate'
]);

const DATA_FLAGS = new Set([
	'-d',
	'--data',
	'--data-raw',
	'--data-binary',
	'--data-ascii',
	'--data-urlencode',
	'--json'
]);

function header(raw: string): CurlRow | null {
	const colon = raw.indexOf(':');
	if (colon <= 0) return null;
	const key = raw.slice(0, colon).trim();
	const value = raw.slice(colon + 1).trim();
	return SECRET_HEADER.test(key)
		? { key, value: '', credential: true }
		: { key, value, credential: false };
}

/** Reads a cURL command into what a one-off card sends. */
export function parseCurl(command: string): CurlRequest {
	const tokens = tokenize(command.trim());
	if (tokens[0]?.toLowerCase() !== 'curl' && tokens[0]?.toLowerCase() !== 'curl.exe') {
		throw new CurlError('not_curl');
	}

	let url = '';
	let method = '';
	let get = false;
	const headers: CurlRow[] = [];
	const data: string[] = [];
	let json = false;
	let skipped = false;
	let basicAuth = false;

	for (let i = 1; i < tokens.length; i++) {
		let token = tokens[i];
		let inline: string | undefined;
		// --flag=value
		if (token.startsWith('--') && token.includes('=')) {
			inline = token.slice(token.indexOf('=') + 1);
			token = token.slice(0, token.indexOf('='));
		}
		const value = () => inline ?? tokens[++i] ?? '';

		if (token === '-X' || token === '--request') method = value().toUpperCase();
		else if (token.startsWith('-X') && token.length > 2) method = token.slice(2).toUpperCase();
		else if (token === '-H' || token === '--header') {
			const row = header(value());
			if (row) headers.push(row);
		} else if (DATA_FLAGS.has(token)) {
			data.push(value());
			if (token === '--json') json = true;
		} else if (token === '-u' || token === '--user') {
			value();
			basicAuth = true;
		} else if (token === '-b' || token === '--cookie') {
			value();
			headers.push({ key: 'Cookie', value: '', credential: true });
		} else if (token === '-A' || token === '--user-agent')
			headers.push({ key: 'User-Agent', value: value(), credential: false });
		else if (token === '--url') url = value();
		else if (token === '-G' || token === '--get') get = true;
		else if (SKIP_WITH_VALUE.has(token)) {
			value();
			skipped = true;
		} else if (token.startsWith('-')) {
			// -s, -L, -k, --compressed, -i, -v: how curl behaves, not what it sends.
			if (
				![
					'-s',
					'--silent',
					'-L',
					'--location',
					'-k',
					'--insecure',
					'--compressed',
					'-i',
					'--include',
					'-v',
					'--verbose',
					'-S',
					'--show-error',
					'-f',
					'--fail'
				].includes(token)
			)
				skipped = true;
		} else if (!url) url = token;
	}

	if (!url) throw new CurlError('no_url');
	if (!/^https?:\/\//i.test(url)) url = `https://${url}`;
	let parsed: URL;
	try {
		parsed = new URL(url);
	} catch {
		throw new CurlError('bad_url');
	}

	const queries: CurlRow[] = [];
	const credentials: string[] = [];
	const addQuery = (key: string, value: string) => {
		if (!key) return;
		queries.push(
			SECRET_QUERY.test(key)
				? { key, value: '', credential: true }
				: { key, value, credential: false }
		);
	};
	parsed.searchParams.forEach((value, key) => addQuery(key, value));
	parsed.search = '';
	parsed.hash = '';

	const dropped: CurlRequest['dropped'] = [];
	let body = '';
	if (get) {
		for (const chunk of data)
			new URLSearchParams(chunk).forEach((value, key) => addQuery(key, value));
	} else if (data.length > 0) {
		const joined = data.join('&');
		try {
			JSON.parse(joined);
			body = joined;
		} catch {
			dropped.push('body_not_json');
		}
	}

	let finalMethod: OneOffMethod = body || (data.length > 0 && !get) ? 'POST' : 'GET';
	if (method === 'GET' || method === 'POST') finalMethod = method as OneOffMethod;
	else if (method) dropped.push('method');
	if (finalMethod === 'GET') body = '';

	if (json) {
		const has = (name: string) => headers.some((row) => row.key.toLowerCase() === name);
		if (!has('content-type'))
			headers.push({ key: 'Content-Type', value: 'application/json', credential: false });
		if (!has('accept'))
			headers.push({ key: 'Accept', value: 'application/json', credential: false });
	}
	if (basicAuth && !headers.some((row) => row.key.toLowerCase() === 'authorization')) {
		headers.push({ key: 'Authorization', value: '', credential: true });
	}
	if (skipped) dropped.push('flags');

	for (const row of [...headers, ...queries]) if (row.credential) credentials.push(row.key);

	return {
		resource: parsed.toString(),
		method: finalMethod,
		queries,
		headers,
		body,
		credentials,
		dropped
	};
}

/**
 * The card's form with the command's request in it. What the command does not
 * say (outputs, title, refresh rate, response format) is left as it was.
 */
export function fillFromCurl(draft: OneOffDraft, request: CurlRequest): OneOffDraft {
	const rows = (list: CurlRow[]) => list.map((row) => ({ ...row, secret: '' }));
	return {
		...draft,
		resource: request.resource,
		method: request.method,
		body: request.body,
		bodySecrets: {},
		queries: rows(request.queries),
		headers: rows(request.headers)
	};
}
