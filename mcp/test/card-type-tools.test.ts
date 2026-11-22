import { afterEach, describe, it } from "node:test";
import assert from "node:assert/strict";

import { fakeBackend, type Backend } from "./support/backend.ts";

import createCardType from "../src/tools/create-card-type.ts";
import listCardTypes from "../src/tools/list-card-types.ts";
import deleteCardType from "../src/tools/delete-card-type.ts";
import tryRequest from "../src/tools/try-request.ts";

const BOARD = "11111111-1111-4111-8111-111111111111";
const KEY = "custom_0123456789abcdef0123456789abcdef";

let backend: Backend | undefined;
afterEach(() => {
    backend?.restore();
    backend = undefined;
});

const input = {
    name: "GitHub repository",
    title: "{{stars}} stars",
    url: "https://api.github.com/repos/$repo",
    response: "json" as const,
    query: { stars: ".stargazers_count" },
    headers: { Authorization: "$token" },
    rate: 600,
    fields: [
        { name: "repo", label: "Repository", type: "text" as const },
        { name: "token", label: "GitHub token", type: "credential" as const },
    ],
};

describe("create_card_type", () => {
    it("sets one up for the person, saying what it will ask for", async () => {
        backend = fakeBackend({
            "POST /api/v1/users/me/postit-templates": {
                status: 201,
                json: { key: KEY, name: "GitHub repository" },
            },
        });

        const out = await createCardType(input);

        assert.match(
            out,
            /Created card type "GitHub repository" \(custom_0123/,
        );
        assert.match(out, /GitHub token \(a credential they pick\)/);
        const sent = backend.to("POST /api/v1/users/me/postit-templates")[0]
            .body as {
            parameters: { key: string; type: string }[];
        };
        assert.deepEqual(
            sent.parameters.map((p) => [p.key, p.type]),
            [
                ["$repo", "string"],
                ["$token", "secret"],
            ],
        );
    });

    it("sets one up for a board", async () => {
        backend = fakeBackend({
            [`POST /api/v1/boards/${BOARD}/postit-templates`]: {
                status: 201,
                json: { key: KEY, name: "GitHub repository" },
            },
        });

        assert.match(
            await createCardType({ ...input, board_id: BOARD }),
            /everyone on that board/,
        );
    });

    it("refuses a request that uses an undeclared field, before asking the board", async () => {
        backend = fakeBackend({});

        const out = await createCardType({
            ...input,
            fields: input.fields.slice(0, 1),
        });

        assert.match(out, /uses \$token, which no field declares/);
        assert.equal(backend.calls.length, 0);
    });

    it("passes the board's explanation of an invalid card type on", async () => {
        backend = fakeBackend({
            "POST /api/v1/users/me/postit-templates": {
                status: 400,
                json: {
                    error: "invalid_postit_template",
                    message: 'output "stars" needs a query',
                },
            },
        });

        await assert.rejects(
            createCardType(input),
            /Refused \(invalid_postit_template\): output "stars" needs a query/,
        );
    });

    it("says what to do at the limit", async () => {
        backend = fakeBackend({
            "POST /api/v1/users/me/postit-templates": {
                status: 409,
                json: { error: "limit_reached" },
            },
        });

        assert.match(await createCardType(input), /delete one first/);
    });
});

describe("list_card_types", () => {
    it("lists the person's own, or what a board may use", async () => {
        const type = {
            key: KEY,
            name: "GitHub repository",
            scope: { kind: "user", owner: "ana" },
            resource: "",
            parameters: [],
            outputs: [],
        };
        backend = fakeBackend({
            "GET /api/v1/users/me/postit-templates": { json: [type] },
            [`GET /api/v1/boards/${BOARD}/postit-templates/usable`]: {
                json: [],
            },
        });

        assert.equal(
            await listCardTypes({}),
            `${KEY} - GitHub repository - the person's own - asks for nothing`,
        );
        assert.equal(
            await listCardTypes({ board_id: BOARD }),
            "No card types yet.",
        );
    });
});

describe("delete_card_type", () => {
    it("deletes it by key", async () => {
        backend = fakeBackend({
            [`DELETE /api/v1/postit-templates/${KEY}`]: { status: 204 },
        });

        assert.equal(
            await deleteCardType({ key: KEY }),
            `Card type ${KEY} deleted`,
        );
    });
});

describe("try_request", () => {
    it("answers the whole response without saving a card", async () => {
        backend = fakeBackend({
            "POST /api/v1/post-its/preview": {
                json: { response: { stargazers_count: 42 } },
            },
        });

        const out = await tryRequest({
            board_id: BOARD,
            url: "https://api.github.com/repos/a/b",
            method: "GET",
            headers: { Accept: "application/json" },
        });

        assert.equal(
            out,
            'https://api.github.com/repos/a/b answered:\n{\n  "stargazers_count": 42\n}',
        );
        assert.deepEqual(backend.to("POST /api/v1/post-its/preview")[0].body, {
            board: BOARD,
            resource: "https://api.github.com/repos/a/b",
            request: {
                method: "GET",
                headers: { Accept: "application/json" },
                queries: {},
            },
            response: "json",
        });
        assert.equal(
            backend.calls.filter((c) => c.path === "/api/v1/post-its").length,
            0,
        );
    });

    it("sends a POST's JSON body as text", async () => {
        backend = fakeBackend({
            "POST /api/v1/post-its/preview": { json: { response: [] } },
        });

        await tryRequest({
            board_id: BOARD,
            url: "https://x.example/search",
            method: "POST",
            body: { q: "cats" },
        });

        const sent = backend.to("POST /api/v1/post-its/preview")[0].body as {
            request: { body: string };
        };
        assert.equal(sent.request.body, '{"q":"cats"}');
    });

    it("says why the request failed", async () => {
        backend = fakeBackend({
            "POST /api/v1/post-its/preview": {
                status: 502,
                json: {
                    error: "The source could not be reached",
                    code: "source_unreachable",
                },
            },
        });

        assert.match(
            await tryRequest({
                board_id: BOARD,
                url: "https://x.example",
                method: "GET",
            }),
            /The request failed: .*could not be reached/,
        );
    });

    it("says when to slow down", async () => {
        backend = fakeBackend({
            "POST /api/v1/post-its/preview": {
                status: 429,
                json: { error: "rate_limited" },
            },
        });

        assert.match(
            await tryRequest({
                board_id: BOARD,
                url: "https://x.example",
                method: "GET",
            }),
            /Too many tries/,
        );
    });
});
