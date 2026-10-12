import Mustache from 'mustache';

/**
 * Card titles may embed the card's outputs as `{{key}}`. Only plain
 * variables are supported: sections, inverted sections, partials and
 * delimiter changes are rejected, so a title can never do more than
 * print values.
 */

const UNSUPPORTED = new Set(['#', '^', '>', '=']);

export type TitleAnalysis = {
	/** Output keys the title uses, in order of appearance, without repeats. */
	used: string[];
	/** Variables that are not an output of the card. */
	unknown: string[];
	/** The title relies on anything but plain `{{name}}`, or does not parse. */
	unsupported: boolean;
	/** The card must fetch its data to render the title. */
	vars: boolean;
};

export function titleVariables(text: string): { names: string[]; unsupported: boolean } {
	let tokens: unknown[][];
	try {
		tokens = Mustache.parse(text) as unknown[][];
	} catch {
		return { names: [], unsupported: true };
	}

	const names: string[] = [];
	let unsupported = false;
	for (const [type, value] of tokens) {
		if (type === 'name' || type === '&') {
			if (!names.includes(value as string)) names.push(value as string);
		} else if (UNSUPPORTED.has(type as string)) {
			unsupported = true;
		}
	}
	return { names, unsupported };
}

export function analyzeTitle(text: string, outputs: string[]): TitleAnalysis {
	const { names, unsupported } = titleVariables(text);
	const used = names.filter((name) => outputs.includes(name));
	const unknown = names.filter((name) => !outputs.includes(name));
	return { used, unknown, unsupported, vars: used.length > 0 && !unsupported };
}

function display(value: unknown): string {
	if (value === null || value === undefined) return '';
	if (Array.isArray(value)) return value.map(display).join(', ');
	if (typeof value === 'object') return JSON.stringify(value);
	return String(value);
}

/**
 * Fills a title with the card's data. The result is plain text: render it
 * with `{...}`, never `{@html}`, as Svelte is the one escaping it.
 */
export function renderTitle(
	text: string,
	data: Record<string, unknown> | null | undefined
): string {
	if (titleVariables(text).unsupported) return text;

	// Own keys only, already stringified: nothing reachable through the
	// prototype (`{{constructor}}`) and no value Mustache could call.
	const view: Record<string, string> = Object.create(null);
	for (const [key, value] of Object.entries(data ?? {})) {
		view[key] = display(value);
	}

	try {
		return Mustache.render(text, view, {}, { escape: (value: unknown) => String(value) });
	} catch {
		return text;
	}
}
