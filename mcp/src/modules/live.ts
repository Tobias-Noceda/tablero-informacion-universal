import type { PostItData, UUID } from "../types/api";

// What a live memory needs: the request a card makes on its own (a one-off
// card, POST /post-its without a well-known) and how to read the answer.
export type LiveMemory = {
    board: UUID;
    title: string;
    url: string;
    response: "json" | "html";
    query: Record<string, string>;
    headers?: Record<string, string>;
    queries?: Record<string, string>;
    rate: number;
};

// A board credential, referenced by name in a header or query value.
const CREDENTIAL = /^\$[A-Z][A-Z0-9_]*$/;

// The body POST /post-its takes for a card with its own request. The backend
// checks it again (validateOneOff): a GET to an http(s) URL, json or html, at
// most 20 headers, query parameters and outputs.
export function liveCardBody(memory: LiveMemory) {
    return {
        board: memory.board,
        title: { text: memory.title, vars: false },
        resource: memory.url,
        request: {
            method: "GET",
            headers: memory.headers ?? {},
            queries: memory.queries ?? {},
        },
        response: memory.response,
        query: memory.query,
        rate: memory.rate,
    };
}

// The credentials a memory's request names, so the agent can tell the user
// which ones the board must hold.
export function credentialsOf(memory: Pick<LiveMemory, "headers" | "queries">) {
    const values = [
        ...Object.values(memory.headers ?? {}),
        ...Object.values(memory.queries ?? {}),
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
