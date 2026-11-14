import type { PostItData, PostItSettings, UUID } from "../types/api";

// What the backend keeps (models.MaxRequestBody), in bytes.
export const MAX_REQUEST_BODY = 16 << 10;

// What a live memory needs: the request a card makes on its own (a one-off
// card, POST /post-its without a well-known) and how to read the answer.
export type LiveMemory = {
    board: UUID;
    title: string;
    url: string;
    method?: "GET" | "POST";
    // A POST's JSON body; a credential in it is a string value "$NAME".
    body?: unknown;
    response: "json" | "html";
    query: Record<string, string>;
    headers?: Record<string, string>;
    queries?: Record<string, string>;
    rate: number;
};

// A board credential, referenced by name in a header, query or body value.
const CREDENTIAL = /^\$[A-Z][A-Z0-9_]*$/;

// A POST's body as the card stores it: JSON text. A GET sends none.
export function requestBody(memory: Pick<LiveMemory, "method" | "body">) {
    if (memory.method !== "POST" || memory.body === undefined) return "";
    return JSON.stringify(memory.body);
}

// The body POST /post-its takes for a card with its own request. The backend
// checks it again (validateOneOff): a GET or POST to an http(s) URL, json or
// html, at most 20 headers, query parameters and outputs, a JSON body of at
// most MAX_REQUEST_BODY bytes.
export function liveCardBody(memory: LiveMemory) {
    const body = requestBody(memory);
    return {
        board: memory.board,
        title: { text: memory.title, vars: false },
        resource: memory.url,
        request: {
            method: memory.method ?? "GET",
            headers: memory.headers ?? {},
            queries: memory.queries ?? {},
            ...(body && { body }),
        },
        response: memory.response,
        query: memory.query,
        rate: memory.rate,
    };
}

// The credentials a memory's request names, so the agent can tell the user
// which ones the board must hold.
export function credentialsOf(
    memory: Pick<LiveMemory, "headers" | "queries" | "method" | "body">,
) {
    const values = [
        ...Object.values(memory.headers ?? {}),
        ...Object.values(memory.queries ?? {}),
        ...(memory.method === "POST" ? stringsOf(memory.body) : []),
    ];
    return [...new Set(values.filter((v) => CREDENTIAL.test(v)).map((v) => v.slice(1)))];
}

const MAX_VALUE = 200;

// What the card read, one output per line, long values cut short.
export function describeData(data: PostItData) {
    const entries = Object.entries(data ?? {});
    if (entries.length === 0) return "(no values)";
    return entries
        .map(([key, value]) => {
            const text = typeof value === "string" ? value : JSON.stringify(value);
            const shown =
                text === undefined
                    ? "(empty)"
                    : text.length > MAX_VALUE
                      ? `${text.slice(0, MAX_VALUE)}…`
                      : text;
            return `${key}: ${shown}`;
        })
        .join("\n");
}

// Every string value in a JSON document, at any depth (object keys excluded).
function stringsOf(node: unknown): string[] {
    if (typeof node === "string") return [node];
    if (Array.isArray(node)) return node.flatMap(stringsOf);
    if (node && typeof node === "object") return Object.values(node).flatMap(stringsOf);
    return [];
}

// How a live memory reads, for an agent about to fix it: only for a card with
// its own request (a one-off card); a card type's request is not the agent's.
export function describeRequest(settings: PostItSettings): string | null {
    if (!settings.resource || settings.wellknown) return null;
    const request = settings.request ?? {};
    const pairs = (record?: Record<string, string> | null) =>
        Object.entries(record ?? {})
            .map(([key, value]) => `${key}: ${value}`)
            .join(", ");
    const lines = [
        `${(request.method || "GET").toUpperCase()} ${settings.resource} (${settings.response || "json"}), read every ${settings.rate} s`,
    ];
    if (pairs(request.queries)) lines.push(`Query string: ${pairs(request.queries)}`);
    if (pairs(request.headers)) lines.push(`Headers: ${pairs(request.headers)}`);
    if (request.body) lines.push(`Body: ${request.body}`);
    const values = Object.entries(settings.query ?? {}).map(([key, expression]) => `${key} ← ${expression}`);
    if (values.length) lines.push(`Values: ${values.join("; ")}`);
    return lines.join("\n");
}
