// A stand-in for the board service: it replaces fetch, answers by route and
// records every request, so a test checks what a tool asked for.

export type Call = {
    method: string;
    path: string;
    body: unknown;
    authorization: string | null;
};

type Answer = { status?: number; json?: unknown };
type Route = (body: unknown, path: string) => Answer;

export type Backend = {
    calls: Call[];
    // The calls whose "METHOD /path" starts with prefix.
    to(prefix: string): Call[];
    restore(): void;
};

// routes maps "METHOD /api/v1/..." to its answer; a path segment written `*`
// matches any value. Anything else answers 404, and is still recorded.
export function fakeBackend(routes: Record<string, Answer | Route>): Backend {
    const original = globalThis.fetch;
    const calls: Call[] = [];

    const match = (method: string, path: string) => {
        for (const [key, answer] of Object.entries(routes)) {
            const [m, pattern] = key.split(" ");
            if (m !== method) continue;
            const want = pattern.split("/");
            const got = path.split("/");
            if (want.length !== got.length) continue;
            if (want.every((part, i) => part === "*" || part === got[i])) {
                return answer;
            }
        }
        return undefined;
    };

    globalThis.fetch = (async (input: string | URL, init: RequestInit = {}) => {
        const url = new URL(String(input));
        const method = init.method ?? "GET";
        const body =
            typeof init.body === "string" ? JSON.parse(init.body) : undefined;
        const headers = new Headers(init.headers);
        calls.push({
            method,
            path: url.pathname,
            body,
            authorization: headers.get("authorization"),
        });

        const route = match(method, url.pathname);
        const answer =
            typeof route === "function" ? route(body, url.pathname) : route;
        const status = answer?.status ?? (answer ? 200 : 404);
        return new Response(
            answer?.json === undefined ? null : JSON.stringify(answer.json),
            {
                status,
                headers: { "Content-Type": "application/json" },
            },
        );
    }) as typeof fetch;

    return {
        calls,
        to: (prefix) =>
            calls.filter((c) => `${c.method} ${c.path}`.startsWith(prefix)),
        restore: () => {
            globalThis.fetch = original;
        },
    };
}
