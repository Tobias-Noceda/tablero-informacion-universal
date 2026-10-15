import { after, before, describe, it } from "node:test";
import assert from "node:assert/strict";
import { createServer, type IncomingMessage } from "node:http";
import type { AddressInfo } from "node:net";

import { boardRoles, claimsOf } from "./auth.js";

function fakeToken(claims: object) {
    const payload = Buffer.from(JSON.stringify(claims)).toString("base64url");
    return `header.${payload}.signature`;
}

const board = "0b6d3b1c-6a7e-4f4e-9d39-8a1f7a7f5e01";

describe("claimsOf", () => {
    it("reads the subject and expiry", () => {
        assert.deepEqual(claimsOf(fakeToken({ sub: "ana", exp: 1700000000 })), {
            sub: "ana",
            exp: 1700000000,
        });
    });

    it("refuses anything that is not a token with both", () => {
        assert.equal(claimsOf(""), null);
        assert.equal(claimsOf("not-a-token"), null);
        assert.equal(claimsOf("a.%%%.c"), null);
        assert.equal(claimsOf(fakeToken({ sub: "ana" })), null);
        assert.equal(claimsOf(fakeToken({ exp: 1 })), null);
    });
});

describe("boardRoles", () => {
    const seen: IncomingMessage[] = [];
    const api = createServer((req, res) => {
        seen.push(req);
        const token = req.headers.authorization?.replace("Bearer ", "");
        const answers: Record<string, [number, object]> = {
            viewer: [200, { id: board, role: "viewer" }],
            odd: [200, { id: board, role: "superuser" }],
            expired: [401, { error: "unauthorized" }],
            stranger: [404, { error: "Board not found" }],
            broken: [500, { error: "boom" }],
        };
        const [status, body] = answers[token ?? ""] ?? [401, {}];
        res.writeHead(status, { "Content-Type": "application/json" });
        res.end(JSON.stringify(body));
    });
    let roleOn: ReturnType<typeof boardRoles>;

    before(async () => {
        await new Promise<void>((resolve) =>
            api.listen(0, "127.0.0.1", resolve),
        );
        const { port } = api.address() as AddressInfo;
        roleOn = boardRoles(`http://127.0.0.1:${port}/api/v1`);
    });

    after(() => new Promise((resolve) => api.close(resolve)));

    it("asks the platform api for the board with the caller's token", async () => {
        assert.equal(await roleOn("viewer", board), "viewer");
        const request = seen.at(-1)!;
        assert.equal(request.method, "GET");
        assert.equal(request.url, `/api/v1/boards/${board}`);
        assert.equal(request.headers.authorization, "Bearer viewer");
    });

    it("gives no role to a token the api refuses", async () => {
        assert.equal(await roleOn("expired", board), null);
    });

    it("gives no role on a board the caller may not see", async () => {
        assert.equal(await roleOn("stranger", board), null);
    });

    it("gives no role when the api fails", async () => {
        assert.equal(await roleOn("broken", board), null);
    });

    it("gives no role it does not know", async () => {
        assert.equal(await roleOn("odd", board), null);
    });

    it("gives no role when the api is unreachable", async () => {
        const closed = createServer();
        await new Promise<void>((resolve) =>
            closed.listen(0, "127.0.0.1", resolve),
        );
        const { port } = closed.address() as AddressInfo;
        await new Promise((resolve) => closed.close(resolve));

        const down = boardRoles(`http://127.0.0.1:${port}/api/v1`);
        assert.equal(await down("viewer", board), null);
    });
});
