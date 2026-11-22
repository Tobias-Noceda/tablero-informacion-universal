// Picking what a card shows from a response tried with "Try" (U2): a click on
// a value of the tree becomes an output, its jq query and a name.

/** Where a value sits in the response: object keys and array indices. */
export type JsonPath = (string | number)[];

const IDENTIFIER = /^[A-Za-z_][A-Za-z0-9_]*$/;

function step(segment: string | number, every: boolean) {
	if (typeof segment === 'number') return every ? '[]' : `[${segment}]`;
	return IDENTIFIER.test(segment) ? `.${segment}` : `[${JSON.stringify(segment)}]`;
}

/**
 * The jq query that reads the value at `path`. With `every`, the index at that
 * position of the path is widened to every element and the results collected
 * in a list: `.data[0].price` becomes `[.data[].price]`.
 */
export function jqPath(path: JsonPath, every?: number): string {
	if (path.length === 0) return '.';
	const body = path.map((segment, i) => step(segment, i === every)).join('');
	const query = body.startsWith('.') ? body : `.${body}`;
	return every === undefined ? query : `[${query}]`;
}

/** The position of the innermost array index in `path`, if any. */
export function lastIndex(path: JsonPath): number | undefined {
	for (let i = path.length - 1; i >= 0; i--) if (typeof path[i] === 'number') return i;
	return undefined;
}

/**
 * A name for the output: the last key of the path as an identifier the card's
 * title can use ({{name}}), not taken by `taken`.
 */
export function outputName(path: JsonPath, taken: string[], every = false): string {
	const key = [...path].reverse().find((segment): segment is string => typeof segment === 'string');
	let base =
		(key ?? 'value')
			.normalize('NFD')
			.replace(/\p{Mn}/gu, '')
			.replace(/[^A-Za-z0-9_]+/g, '_')
			.replace(/^_+|_+$/g, '')
			.toLowerCase() || 'value';
	if (/^[0-9]/.test(base)) base = `v_${base}`;
	if (every) base = `${base}_list`;

	const used = new Set(taken);
	if (!used.has(base)) return base;
	for (let n = 2; ; n++) if (!used.has(`${base}_${n}`)) return `${base}_${n}`;
}

/** How many children of an object or array the tree shows before "N more". */
export const TREE_CHILDREN = 50;

/** The children the tree shows of an object or array, and how many it hides. */
export function childrenOf(value: unknown): {
	entries: [string | number, unknown][];
	hidden: number;
} {
	const all: [string | number, unknown][] = Array.isArray(value)
		? value.map((item, i) => [i, item])
		: value !== null && typeof value === 'object'
			? Object.entries(value)
			: [];
	return { entries: all.slice(0, TREE_CHILDREN), hidden: Math.max(0, all.length - TREE_CHILDREN) };
}

/** A leaf as the tree shows it: strings quoted and cut, the rest as JSON. */
export function leafText(value: unknown, max = 80): string {
	const text = JSON.stringify(value) ?? String(value);
	return text.length > max ? `${text.slice(0, max - 1)}…` : text;
}
