import { afterEach, describe, it } from "node:test";
import assert from "node:assert/strict";

import { fakeBackend, type Backend } from "./support/backend.ts";
import * as api from "../src/modules/api.ts";

let backend: Backend | undefined;
afterEach(() => {
    backend?.restore();
    backend = undefined;
});

// What the agent reads when the board service refuses: it has to tell the
// person what to do, not just the status.
describe("api errors", () => {
    const refusing = (status: number, json: object) =>
        fakeBackend({ "GET /api/v1/boards": { status, json } });

    it("explains that the MCP server needs a paid plan", async () => {
        backend = refusing(402, { error: "plan_limit", limit: "api_access" });
        await assert.rejects(
            api.get("/v1/boards"),
            /requires a Pro or Max plan/,
        );
    });

    it("names the plan limit that was reached", async () => {
        backend = refusing(402, { error: "plan_limit", limit: "nodes" });
        await assert.rejects(
            api.get("/v1/boards"),
            /Plan limit reached \(nodes\)/,
        );
    });

    it("asks for a new token when it is invalid or expired", async () => {
        backend = refusing(401, { error: "unauthorized" });
        await assert.rejects(
            api.get("/v1/boards"),
            /Create a new one in your profile/,
        );
    });

    it("says what an access token may do when a route is closed to it", async () => {
        backend = refusing(403, { error: "token_not_allowed" });
        await assert.rejects(
            api.get("/v1/boards"),
            /can only read and edit boards and their cards/,
        );
    });

    it("does not tell a missing board from one the caller cannot see", async () => {
        backend = refusing(404, { error: "Not found" });
        await assert.rejects(
            api.get("/v1/boards"),
            /Not found, or you have no access to it/,
        );
    });

    it("reports an unreachable board service without the network details", async () => {
        const original = globalThis.fetch;
        globalThis.fetch = (async () => {
            throw new TypeError("connect ECONNREFUSED 10.0.0.5:31126");
        }) as typeof fetch;
        try {
            await assert.rejects(api.get("/v1/boards"), (err: Error) => {
                assert.match(err.message, /unable to reach the board service/);
                assert.doesNotMatch(err.message, /10\.0\.0\.5/);
                return true;
            });
        } finally {
            globalThis.fetch = original;
        }
    });
});
