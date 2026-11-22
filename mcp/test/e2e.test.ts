import { describe, it } from "node:test";
import assert from "node:assert/strict";

// End to end against a running MCP server, as a client would talk to it
// (JSON-RPC over streamable HTTP), through the real backend. Skipped unless
// both are set:
//   MCP_E2E_URL    the server's /mcp endpoint, e.g. http://localhost/mcp
//   MCP_E2E_TOKEN  a personal access token (tiu_pat_…) of a Pro or Max account
// It creates its own board and deletes it at the end.
const URL_ = process.env.MCP_E2E_URL;
const TOKEN = process.env.MCP_E2E_TOKEN;

const TOOLS = [
    "associate",
    "consolidate",
    "create_card_type",
    "create_pkg",
    "delete_card_type",
    "delete_pkg",
    "dissociate",
    "forget",
    "list_card_types",
    "list_pkgs",
    "memories",
    "perceive",
    "reconsolidate",
    "retrieve",
    "try_request",
];

// A source the backend can reach in the Docker stack started with
// perf/docker-compose.perf.yaml (its SSRF guard lets 203.0.113.0/24 through).
const SOURCE =
    process.env.MCP_E2E_SOURCE ?? "http://203.0.113.10:8080/value?delay=0";

type RpcResult = {
    result?: {
        tools?: { name: string }[];
        content?: { type: string; text: string }[];
        isError?: boolean;
    };
    error?: { message: string };
};

class Client {
    #id = 0;
    #session: string | null = null;

    async #post(body: object): Promise<Response> {
        const res = await fetch(URL_!, {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
                Accept: "application/json, text/event-stream",
                Authorization: `Bearer ${TOKEN}`,
                ...(this.#session && { "mcp-session-id": this.#session }),
            },
            body: JSON.stringify(body),
        });
        this.#session = res.headers.get("mcp-session-id") ?? this.#session;
        return res;
    }

    // One request, its answer read from JSON or from the event stream.
    async rpc(method: string, params: object = {}): Promise<RpcResult> {
        const res = await this.#post({
            jsonrpc: "2.0",
            id: ++this.#id,
            method,
            params,
        });
        assert.equal(res.ok, true, `${method}: HTTP ${res.status}`);
        const text = await res.text();
        if (
            (res.headers.get("content-type") ?? "").includes(
                "text/event-stream",
            )
        ) {
            const data = text
                .split("\n")
                .filter((line) => line.startsWith("data:"))
                .map((line) => line.slice(5).trim())
                .pop();
            return JSON.parse(data ?? "{}");
        }
        return JSON.parse(text);
    }

    async notify(method: string) {
        await this.#post({ jsonrpc: "2.0", method });
    }

    // A tool's text answer; a tool error fails the test with its message.
    async call(name: string, args: object): Promise<string> {
        const answer = await this.rpc("tools/call", { name, arguments: args });
        assert.equal(
            answer.error,
            undefined,
            `${name}: ${answer.error?.message}`,
        );
        const text =
            answer.result?.content?.map((c) => c.text).join("\n") ?? "";
        assert.notEqual(answer.result?.isError, true, `${name}: ${text}`);
        return text;
    }
}

const uuid = /[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/;

describe("MCP server end to end", { skip: !URL_ || !TOKEN }, () => {
    it("keeps an agent's memories on a board it creates and removes", async () => {
        const client = new Client();
        const init = await client.rpc("initialize", {
            protocolVersion: "2025-06-18",
            capabilities: {},
            clientInfo: { name: "tiu-e2e", version: "1" },
        });
        assert.equal(init.error, undefined);
        await client.notify("notifications/initialized");

        const listed = await client.rpc("tools/list");
        assert.deepEqual(
            listed.result?.tools?.map((t) => t.name).sort(),
            TOOLS,
        );

        const created = await client.call("create_pkg", {
            name: `e2e ${new Date().toISOString().slice(0, 19)}`,
        });
        const board = created.match(uuid)?.[0];
        assert.ok(board, created);

        try {
            const first = (
                await client.call("consolidate", {
                    board_id: board,
                    title: "The deploy runs on Vercel",
                    description: "Production is **corcheria.company**.",
                    relevant: [],
                    position: { x: 0, y: 0 },
                })
            ).match(uuid)?.[0];
            const second = (
                await client.call("consolidate", {
                    board_id: board,
                    title: "Migrations run on deploy",
                    description: "Only for production.",
                    relevant: [],
                    position: { x: 300, y: 0 },
                })
            ).match(uuid)?.[0];
            assert.ok(first && second);

            assert.match(
                await client.call("associate", {
                    board_id: board,
                    id: second,
                    relevant: [first],
                    label: "part of",
                    directed: true,
                }),
                /1 new/,
            );

            const recalled = await client.call("retrieve", {
                board_id: board,
                postit_id: second,
            });
            assert.match(recalled, /Migrations run on deploy/);
            assert.match(recalled, new RegExp(`→ "part of" ${first}`));

            await client.call("reconsolidate", {
                id: first,
                title: "The deploy runs on Vercel and Docker",
            });
            assert.match(
                await client.call("memories", { id: board }),
                /Vercel and Docker/,
            );

            await client.call("dissociate", {
                board_id: board,
                memory_id_1: first,
                memory_id_2: second,
            });
            assert.match(
                await client.call("retrieve", {
                    board_id: board,
                    postit_id: second,
                }),
                /\(none\)/,
            );

            // A card type for the person: tried first, then set up, listed
            // and removed.
            assert.match(
                await client.call("try_request", {
                    board_id: board,
                    url: SOURCE,
                }),
                /"value":/,
            );
            const made = await client.call("create_card_type", {
                board_id: board,
                name: "e2e value",
                title: "Value: {{value}}",
                url: SOURCE.split("?")[0],
                queries: { delay: "$delay" },
                query: { value: ".value" },
                fields: [
                    {
                        name: "delay",
                        label: "Delay (ms)",
                        type: "number",
                        default: "0",
                    },
                ],
            });
            const key = made.match(/custom_[0-9a-f]{32}/)?.[0];
            assert.ok(key, made);
            assert.match(
                await client.call("list_card_types", { board_id: board }),
                new RegExp(
                    String.raw`${key} - e2e value - this board's - asks for Delay \(ms\) \(number\)`,
                ),
            );
            assert.match(
                await client.call("delete_card_type", { key }),
                /deleted/,
            );

            await client.call("forget", { id: second });
            assert.doesNotMatch(
                await client.call("memories", { id: board }),
                new RegExp(second!),
            );
        } finally {
            await client.call("delete_pkg", { id: board });
        }

        assert.doesNotMatch(
            await client.call("list_pkgs", {}),
            new RegExp(board),
        );
    });
});
