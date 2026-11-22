import { afterEach, describe, it } from "node:test";
import assert from "node:assert/strict";

import { z } from "zod";

import { fakeBackend, type Backend } from "./support/backend.ts";
// @ts-expect-error: a plain JavaScript stand-in, resolved by test/support/hooks.mjs
import { setAuthorization } from "xmcp/headers";

import listPkgs from "../src/tools/list-pkgs.ts";
import createPkg from "../src/tools/create-pkg.ts";
import deletePkg from "../src/tools/delete-pkgs.ts";
import memories from "../src/tools/memories.ts";
import consolidate, {
    schema as consolidateSchema,
} from "../src/tools/consolidate.ts";
import reconsolidate from "../src/tools/reconsolidate.ts";
import forget from "../src/tools/forget.ts";
import associate from "../src/tools/associate.ts";
import dissociate from "../src/tools/dissociate.ts";
import retrieve from "../src/tools/retrieve.ts";
import perceive, { schema as perceiveSchema } from "../src/tools/perceive.ts";

const BOARD = "11111111-1111-4111-8111-111111111111";
const A = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa";
const B = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb";
const C = "cccccccc-cccc-4ccc-8ccc-cccccccccccc";
const ORG = "99999999-9999-4999-8999-999999999999";

const card = (id: string, text: string, vars = false, x = 0, y = 0) => ({
    id,
    type: "static_card",
    title: { text, vars },
    position: { x, y },
});

let backend: Backend | undefined;
afterEach(() => {
    backend?.restore();
    backend = undefined;
    setAuthorization("Bearer tiu_pat_test");
});

describe("list_pkgs", () => {
    it("lists every board with its organization, sending the caller's token", async () => {
        setAuthorization("Bearer tiu_pat_ana");
        backend = fakeBackend({
            "GET /api/v1/boards": {
                json: [
                    { id: BOARD, name: "Work", org: ORG },
                    { id: A, name: "Mine" },
                ],
            },
            "GET /api/v1/orgs": { json: [{ id: ORG, name: "Acme" }] },
        });

        const out = await listPkgs();

        assert.equal(out, `${BOARD} - Acme - Work\n${A} - Personal - Mine`);
        assert.ok(
            backend.calls.every(
                (c) => c.authorization === "Bearer tiu_pat_ana",
            ),
        );
    });
});

describe("create_pkg", () => {
    it("creates a personal board", async () => {
        backend = fakeBackend({
            "POST /api/v1/boards": {
                status: 201,
                json: { id: BOARD, name: "Ideas" },
            },
        });

        const out = await createPkg({ name: "Ideas" });

        assert.equal(out, `Created board Ideas (${BOARD})`);
        assert.deepEqual(backend.to("POST /api/v1/boards")[0].body, {
            name: "Ideas",
        });
    });

    it("finds the organization by name, whatever the case", async () => {
        backend = fakeBackend({
            "GET /api/v1/orgs": { json: [{ id: ORG, name: "Acme" }] },
            "POST /api/v1/boards": {
                status: 201,
                json: { id: BOARD, name: "Plans" },
            },
        });

        await createPkg({ name: "Plans", organization: "acme" });

        assert.deepEqual(backend.to("POST /api/v1/boards")[0].body, {
            name: "Plans",
            org: ORG,
        });
    });

    it("refuses an organization the caller is not in, creating nothing", async () => {
        backend = fakeBackend({ "GET /api/v1/orgs": { json: [] } });

        await assert.rejects(
            createPkg({ name: "Plans", organization: "Globex" }),
            /not in an organization called "Globex"/,
        );
        assert.equal(backend.to("POST").length, 0);
    });
});

describe("delete_pkg and forget", () => {
    it("delete the board and the card they name", async () => {
        backend = fakeBackend({
            [`DELETE /api/v1/boards/${BOARD}`]: { status: 204 },
            [`DELETE /api/v1/post-its/${A}`]: { json: [] },
        });

        assert.equal(await deletePkg({ id: BOARD }), "PKG deleted");
        assert.equal(await forget({ id: A }), "Memory deleted");
        assert.deepEqual(
            backend.calls.map((c) => `${c.method} ${c.path}`),
            [`DELETE /api/v1/boards/${BOARD}`, `DELETE /api/v1/post-its/${A}`],
        );
    });
});

describe("memories", () => {
    it("lists a board's memories, filling in titles that read their data", async () => {
        backend = fakeBackend({
            [`GET /api/v1/boards/${BOARD}`]: {
                json: {
                    id: BOARD,
                    postits: [
                        card(A, "Plain", false, 1, 2),
                        card(B, "Price: {{usd}}", true, 3, 4),
                    ],
                    strands: [],
                },
            },
            [`GET /api/v1/post-its/${B}`]: { json: { usd: "42" } },
        });

        const out = await memories({ id: BOARD });

        assert.equal(out, `${A} - (1; 2) - Plain\n${B} - (3; 4) - Price: 42`);
    });
});

describe("consolidate", () => {
    it("saves a note where it is told and links it to the related memories", async () => {
        backend = fakeBackend({
            "POST /api/v1/post-its": { status: 201, json: { id: C } },
            [`PATCH /api/v1/post-its/${C}/position`]: { status: 204 },
            [`POST /api/v1/boards/${BOARD}/strands`]: { status: 201, json: {} },
        });

        const out = await consolidate({
            board_id: BOARD,
            title: "Fact",
            description: "**bold**",
            relevant: [A, B],
            position: { x: 10, y: 20 },
        });

        assert.equal(out, `Created memory ${C}`);
        assert.deepEqual(backend.to("POST /api/v1/post-its")[0].body, {
            board: BOARD,
            title: { text: "Fact", vars: false },
            well_known: "static_card",
            params: { text: "**bold**" },
        });
        assert.deepEqual(backend.to("PATCH")[0].body, { x: 10, y: 20 });
        assert.deepEqual(
            backend.to("POST /api/v1/boards").map((c) => c.body),
            [
                { source: C, target: A },
                { source: C, target: B },
            ],
        );
    });

    it("refuses what is not a board id before anything is sent", () => {
        const parsed = z.object(consolidateSchema).safeParse({
            board_id: "not-a-uuid",
            title: "x",
            description: "y",
            relevant: [],
            position: { x: 0, y: 0 },
        });
        assert.equal(parsed.success, false);
    });
});

describe("reconsolidate", () => {
    it("changes only what it is given", async () => {
        backend = fakeBackend({
            [`PATCH /api/v1/post-its/${A}/title`]: { status: 204 },
            [`PATCH /api/v1/post-its/${A}/settings`]: { status: 204 },
        });

        await reconsolidate({ id: A, title: "New title" });
        assert.deepEqual(
            backend.calls.map((c) => [c.path, c.body]),
            [
                [
                    `/api/v1/post-its/${A}/title`,
                    { title: { text: "New title", vars: false } },
                ],
            ],
        );

        backend.calls.length = 0;
        await reconsolidate({ id: A, description: "New text" });
        assert.deepEqual(
            backend.calls.map((c) => [c.path, c.body]),
            [
                [
                    `/api/v1/post-its/${A}/settings`,
                    { params: { text: "New text" } },
                ],
            ],
        );

        backend.calls.length = 0;
        await reconsolidate({ id: A });
        assert.equal(backend.calls.length, 0);
    });
});

describe("associate", () => {
    const boardWith = (strands: object[]) => ({
        [`GET /api/v1/boards/${BOARD}`]: {
            json: { id: BOARD, postits: [], strands },
        },
        [`POST /api/v1/boards/${BOARD}/strands`]: { status: 201, json: {} },
        [`PATCH /api/v1/boards/${BOARD}/strands/*`]: { json: {} },
    });

    it("draws new strands with the label, pointing away when directed, once per memory", async () => {
        backend = fakeBackend(boardWith([]));

        const out = await associate({
            board_id: BOARD,
            id: A,
            relevant: [B, B],
            label: "depends on",
            directed: true,
        });

        assert.match(out, /\(1 new, 0 updated\)/);
        assert.deepEqual(
            backend.to("POST").map((c) => c.body),
            [{ source: A, target: B, label: "depends on", arrow: "end" }],
        );
    });

    it("restyles a strand drawn the other way, pointing it away from this memory", async () => {
        backend = fakeBackend(boardWith([{ id: "s1", source: B, target: A }]));

        const out = await associate({
            board_id: BOARD,
            id: A,
            relevant: [B],
            directed: true,
        });

        assert.match(out, /\(0 new, 1 updated\)/);
        const [patch] = backend.to("PATCH");
        assert.equal(patch.path, `/api/v1/boards/${BOARD}/strands/s1`);
        assert.deepEqual(patch.body, { arrow: "start" });
        assert.equal(backend.to("POST").length, 0);
    });

    it("leaves an existing strand alone when nothing about its style is given", async () => {
        backend = fakeBackend(
            boardWith([{ id: "s1", source: A, target: B, label: "kept" }]),
        );

        await associate({ board_id: BOARD, id: A, relevant: [B] });

        assert.equal(backend.to("PATCH").length + backend.to("POST").length, 0);
    });
});

describe("dissociate", () => {
    it("removes the strands between the two memories, either way, and no others", async () => {
        backend = fakeBackend({
            [`GET /api/v1/boards/${BOARD}`]: {
                json: {
                    id: BOARD,
                    postits: [],
                    strands: [
                        { id: "ab", source: A, target: B },
                        { id: "ba", source: B, target: A },
                        { id: "ac", source: A, target: C },
                    ],
                },
            },
            [`DELETE /api/v1/boards/${BOARD}/strands/*`]: { status: 204 },
        });

        await dissociate({ board_id: BOARD, memory_id_1: A, memory_id_2: B });

        assert.deepEqual(
            backend
                .to("DELETE")
                .map((c) => c.path.split("/").pop())
                .sort(),
            ["ab", "ba"],
        );
    });
});

describe("retrieve", () => {
    it("shows the memory's data, how a live one reads and how the related ones relate", async () => {
        backend = fakeBackend({
            [`GET /api/v1/boards/${BOARD}`]: {
                json: {
                    id: BOARD,
                    postits: [
                        card(A, "BTC: {{usd}}", true),
                        card(B, "Wallet"),
                        card(C, "Unrelated"),
                    ],
                    strands: [
                        {
                            id: "s",
                            source: A,
                            target: B,
                            arrow: "end",
                            label: "funds",
                        },
                    ],
                },
            },
            [`GET /api/v1/post-its/${A}`]: { json: { usd: "90000" } },
            [`GET /api/v1/post-its/${A}/settings`]: {
                json: {
                    id: A,
                    board: BOARD,
                    params: {},
                    rate: 60,
                    resource: "https://api.example.com/price",
                    request: {
                        method: "GET",
                        headers: { Authorization: "$KEY" },
                        queries: {},
                    },
                    response: "json",
                    query: { usd: ".usd" },
                },
            },
        });

        const out = await retrieve({ board_id: BOARD, postit_id: A });

        assert.match(out, /^"BTC: 90000"/);
        assert.match(out, /Data: \{"usd":"90000"\}/);
        assert.match(out, /https:\/\/api\.example\.com\/price/);
        assert.match(out, /\$KEY/);
        assert.match(out, new RegExp(`→ "funds" ${B} - \\(0; 0\\) - Wallet`));
        assert.doesNotMatch(out, /Unrelated/);
    });

    it("refuses a memory that is not on the board", async () => {
        backend = fakeBackend({
            [`GET /api/v1/boards/${BOARD}`]: {
                json: { id: BOARD, postits: [], strands: [] },
            },
        });

        await assert.rejects(
            retrieve({ board_id: BOARD, postit_id: A }),
            /not on this board/,
        );
    });
});

describe("perceive", () => {
    const memory = {
        board_id: BOARD,
        title: "BTC: {{usd}}",
        url: "https://api.example.com/price",
        method: "GET" as const,
        response: "json" as const,
        query: { usd: ".usd" },
        headers: { Authorization: "$KEY" },
        rate: 300,
        relevant: [] as string[],
        position: { x: 5, y: 6 },
    };

    it("creates a live card, checks that it reads and places it", async () => {
        backend = fakeBackend({
            "POST /api/v1/post-its": { status: 201, json: { id: C } },
            [`GET /api/v1/post-its/${C}`]: { json: { usd: "90000" } },
            [`PATCH /api/v1/post-its/${C}/position`]: { status: 204 },
        });

        await perceive(memory);

        const created = backend.to("POST /api/v1/post-its")[0].body as Record<
            string,
            unknown
        >;
        assert.equal(created.resource, memory.url);
        assert.deepEqual(created.query, { usd: ".usd" });
        assert.deepEqual((created.request as Record<string, unknown>).headers, {
            Authorization: "$KEY",
        });
        assert.deepEqual(backend.to("PATCH")[0].body, { x: 5, y: 6 });
    });

    it("does not keep a card that cannot read its address, and names the credentials it needs", async () => {
        backend = fakeBackend({
            "POST /api/v1/post-its": { status: 201, json: { id: C } },
            [`GET /api/v1/post-its/${C}`]: {
                status: 502,
                json: {
                    error: "Resource returned 401",
                    code: "source_status",
                    status: 401,
                },
            },
            [`DELETE /api/v1/post-its/${C}`]: { json: [] },
        });

        const out = await perceive(memory);

        assert.match(
            out,
            /could not read https:\/\/api\.example\.com\/price, so it was not kept/,
        );
        assert.match(out, /KEY/);
        assert.equal(backend.to("DELETE").length, 1);
        assert.equal(backend.to("PATCH").length, 0);
    });

    it("refuses a body on a GET without sending anything", async () => {
        backend = fakeBackend({});

        const out = await perceive({ ...memory, body: { q: 1 } });

        assert.match(out, /GET request carries no body/);
        assert.equal(backend.calls.length, 0);
    });

    it("only reads http(s) addresses", () => {
        const parsed = z
            .object(perceiveSchema)
            .safeParse({ ...memory, url: "file:///etc/passwd" });
        assert.equal(parsed.success, false);
    });
});
