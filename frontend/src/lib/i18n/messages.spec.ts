import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

type Messages = { [key: string]: string | Messages };

function load(locale: string): Messages {
	const raw = readFileSync(new URL(`../../../messages/${locale}.json`, import.meta.url), 'utf-8');
	const messages = JSON.parse(raw);
	delete messages.$schema;
	return messages;
}

// Every message by its dotted key, as `m['group.key']` reads it.
function flatten(messages: Messages, prefix = ''): Map<string, string> {
	const out = new Map<string, string>();
	for (const [key, value] of Object.entries(messages)) {
		if (typeof value === 'string') out.set(prefix + key, value);
		else for (const [inner, text] of flatten(value, `${prefix}${key}.`)) out.set(inner, text);
	}
	return out;
}

const variables = (text: string) =>
	[...text.matchAll(/\{(\w+)\}/g)].map((match) => match[1]).sort();

const en = flatten(load('en'));
const es = flatten(load('es'));

describe('messages', () => {
	it('has every key in both languages', () => {
		expect([...es.keys()].filter((key) => !en.has(key))).toEqual([]);
		expect([...en.keys()].filter((key) => !es.has(key))).toEqual([]);
	});

	it('keeps the same placeholders in a translation', () => {
		const mismatched = [...en].filter(
			([key, text]) => es.has(key) && variables(text).join() !== variables(es.get(key)!).join()
		);
		expect(mismatched.map(([key]) => key)).toEqual([]);
	});

	it('leaves no message empty', () => {
		const empty = [...en, ...es].filter(([, text]) => text.trim() === '');
		expect(empty.map(([key]) => key)).toEqual([]);
	});
});
