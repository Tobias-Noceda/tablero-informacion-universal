// Card types an agent sets up for its person or a board (post-it templates):
// the agent knows the API, the person then only fills in the fields, picking
// their own credential where the type asks for one. A card type never holds a
// secret: a credential field is an empty slot, filled when a card is made.

// The fields a card type asks for, in the agent's words, and the backend's.
export const FIELD_TYPES = {
    text: "string",
    number: "number",
    date: "date",
    datetime: "datetime",
    credential: "secret",
} as const;

export type FieldType = keyof typeof FIELD_TYPES;

export type CardTypeField = {
    name: string;
    label: string;
    type: FieldType;
    placeholder?: string;
    default?: string;
};

export type CardTypeInput = {
    name: string;
    description?: string;
    title: string;
    url: string;
    response: "json" | "html";
    query: Record<string, string>;
    labels?: Record<string, string>;
    queries?: Record<string, string>;
    headers?: Record<string, string>;
    rate: number;
    fields: CardTypeField[];
};

export type CardType = {
    key: string;
    name: string;
    description?: string;
    scope: { kind: "user" | "board" | "org"; owner: string };
    resource: string;
    parameters: { key: string; label: string; type: string }[] | null;
    outputs: { key: string; label: string }[] | null;
    usable?: boolean;
};

export class CardTypeError extends Error {
    constructor(message: string) {
        super(message);
        this.name = "CardTypeError";
    }
}

const FIELD_NAME = /^[a-z][a-z0-9_]*$/;
const USED_FIELD = /\$[a-z][a-z0-9_]*/g;

/** A field's placeholder: "repo" or "$repo" is "$repo". */
export function placeholder(name: string): string {
    const bare = name.trim().replace(/^\$/, "");
    if (!FIELD_NAME.test(bare)) {
        throw new CardTypeError(
            `The field name "${name}" must be lowercase letters, digits and _ (starting with a letter), like "repo" or "api_key".`,
        );
    }
    return `$${bare}`;
}

/** The $fields the request uses that the card type does not declare. */
export function undeclared(input: CardTypeInput): string[] {
    const declared = new Set(
        input.fields.map((field) => placeholder(field.name)),
    );
    const used = [
        input.url,
        ...Object.values(input.queries ?? {}),
        ...Object.values(input.headers ?? {}),
    ].flatMap((value) => value.match(USED_FIELD) ?? []);
    return [...new Set(used.filter((name) => !declared.has(name)))];
}

/** What POST /…/postit-templates takes for the card type. */
export function cardTypeBody(input: CardTypeInput) {
    const outputs = Object.keys(input.query);
    const vars = outputs.some((key) =>
        new RegExp(`\\{\\{\\s*${key}\\s*\\}\\}`).test(input.title),
    );
    return {
        name: input.name,
        ...(input.description && { description: input.description }),
        title: { text: input.title, vars },
        resource: input.url,
        request: {
            method: "GET",
            headers: input.headers ?? {},
            queries: input.queries ?? {},
        },
        response: input.response,
        query: input.query,
        rate: input.rate,
        parameters: input.fields.map((field) => ({
            key: placeholder(field.name),
            label: field.label,
            type: FIELD_TYPES[field.type],
            ...(field.placeholder && { placeholder: field.placeholder }),
            ...(field.default &&
                field.type !== "credential" && { default: field.default }),
        })),
        outputs: outputs.map((key) => ({
            key,
            label: input.labels?.[key] ?? key,
        })),
    };
}

const FIELD_WORDS: Record<string, FieldType> = Object.fromEntries(
    Object.entries(FIELD_TYPES).map(([word, type]) => [
        type,
        word as FieldType,
    ]),
);

/** One line per card type: key, name, where it lives and what it asks for. */
export function describeCardType(type: CardType): string {
    const where =
        type.scope.kind === "user"
            ? "the person's own"
            : type.scope.kind === "board"
              ? "this board's"
              : "the organization's";
    const asks = (type.parameters ?? []).map(
        (param) => `${param.label} (${FIELD_WORDS[param.type] ?? param.type})`,
    );
    const shows = (type.outputs ?? []).map((output) => output.key);
    return [
        `${type.key} - ${type.name} - ${where}`,
        asks.length ? `asks for ${asks.join(", ")}` : "asks for nothing",
        shows.length ? `shows ${shows.join(", ")}` : null,
        type.usable === false ? "not usable here (made by someone else)" : null,
    ]
        .filter(Boolean)
        .join(" - ");
}

const MAX_RESPONSE = 6000;

/** A tried response for the agent to pick its values from, cut when long. */
export function describeResponse(value: unknown): string {
    if (value === undefined || value === null) return "(empty response)";
    const text = JSON.stringify(value, null, 2);
    return text.length > MAX_RESPONSE
        ? `${text.slice(0, MAX_RESPONSE)}\n… (cut: ${text.length} characters in all)`
        : text;
}
