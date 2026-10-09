import { afterEach, beforeEach, describe, it } from "node:test";
import assert from "node:assert/strict";
import { createServer, type Server } from "node:http";
import type { AddressInfo } from "node:net";
import { randomUUID } from "node:crypto";

import { io, type Socket } from "socket.io-client";

import type { BoardRole } from "./auth.js";
import { attach, type Presence, type Realtime } from "./socket.js";

const board = "0b6d3b1c-6a7e-4f4e-9d39-8a1f7a7f5e01";

// A token valid for `seconds` (fractions allowed, so timers stay short).
function token(sub: string, seconds = 900) {
    const claims = { sub, exp: (Date.now() + seconds * 1000) / 1000 };
    const payload = Buffer.from(JSON.stringify(claims)).toString("base64url");
    return `header.${payload}.${randomUUID()}`;
}

class MemoryPresence implements Presence {
    boards = new Map<string, Map<string, string>>();
    left: string[] = [];

    async join(board: string, peer: string, user: string) {
        const peers = this.boards.get(board) ?? new Map<string, string>();
        this.boards.set(board, peers);
        const before = Object.fromEntries(peers);
        peers.set(peer, user);
        return before;
    }

    async leave(board: string, peer: string) {
        this.boards.get(board)?.delete(peer);
        this.left.push(peer);
    }
}

const once = <T>(socket: Socket, event: string) =>
    new Promise<T>((resolve) => socket.once(event, resolve));

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

describe("board sockets", () => {
    let server: Server;
    let realtime: Realtime;
    let presence: MemoryPresence;
    let url: string;
    // What the platform API would answer for each token.
    let roles: Map<string, BoardRole>;
    const clients: Socket[] = [];

    function connect(
        auth: object,
        query: object = { board, peer: randomUUID() },
    ) {
        const client = io(url, {
            path: "/ws",
            transports: ["websocket"],
            reconnection: false,
            auth,
            query: query as Record<string, string>,
        });
        clients.push(client);
        return client;
    }

    async function member(
        sub: string,
        role: BoardRole = "viewer",
        seconds?: number,
    ) {
        const credential = token(sub, seconds);
        roles.set(credential, role);
        const client = connect({ token: credential });
        const peers = await once<Record<string, string>>(client, "peers");
        return { client, peers };
    }

    beforeEach(async () => {
        roles = new Map();
        presence = new MemoryPresence();
        server = createServer();
        realtime = attach(server, {
            roleOn: async (token, on) =>
                on === board ? (roles.get(token) ?? null) : null,
            presence,
            renewLead: 500,
            grace: 100,
        });
        await new Promise<void>((resolve) =>
            server.listen(0, "127.0.0.1", resolve),
        );
        url = `http://127.0.0.1:${(server.address() as AddressInfo).port}`;
    });

    afterEach(async () => {
        clients.splice(0).forEach((client) => client.close());
        await realtime.close();
    });

    it("refuses a connection without a token", async () => {
        const error = await once<Error>(connect({}), "connect_error");
        assert.equal(error.message, "unauthorized");
    });

    it("refuses a token the platform api does not accept", async () => {
        const error = await once<Error>(
            connect({ token: token("mallory") }),
            "connect_error",
        );
        assert.equal(error.message, "unauthorized");
    });

    it("refuses a malformed board or peer", async () => {
        const credential = token("ana");
        roles.set(credential, "owner");
        const error = await once<Error>(
            connect(
                { token: credential },
                { board: "nope", peer: randomUUID() },
            ),
            "connect_error",
        );
        assert.equal(error.message, "invalid_handshake");
    });

    it("lets a member in, with the peers already there by peer id", async () => {
        const ana = await member("ana", "owner");
        assert.deepEqual(ana.peers, {});

        const bob = await member("bob");
        const [peer, user] = Object.entries(bob.peers)[0]!;
        assert.equal(user, "ana");
        assert.equal(presence.boards.get(board)?.get(peer), "ana");
    });

    it("keeps two tabs of the same user apart", async () => {
        await member("ana");
        await member("ana");
        assert.equal(presence.boards.get(board)?.size, 2);
    });

    it("relays board updates to the members", async () => {
        const { client } = await member("bob");
        const update = once<{ board: { name: string } }>(client, "update");
        realtime.notify(board.toUpperCase(), { name: "Roadmap" });
        assert.equal((await update).board.name, "Roadmap");
    });

    it("forgets the peer when it disconnects", async () => {
        const { client } = await member("bob");
        client.close();
        await sleep(100);
        assert.equal(presence.boards.get(board)?.size, 0);
        assert.equal(presence.left.length, 1);
    });

    it("asks for a fresh token before the current one expires and keeps going", async () => {
        const { client } = await member("bob", "viewer", 1);
        await once(client, "token_expiring");

        const fresh = token("bob");
        roles.set(fresh, "viewer");
        client.emit("auth", { token: fresh });

        await sleep(1000);
        assert.equal(client.connected, true);
    });

    it("drops a socket that never renews", async () => {
        const { client } = await member("bob", "viewer", 0.6);
        const reason = await once<string>(client, "disconnect");
        assert.equal(reason, "io server disconnect");
    });

    it("drops a socket whose user lost the board at renewal", async () => {
        const { client } = await member("bob", "viewer", 1);
        await once(client, "token_expiring");

        const disconnected = once<string>(client, "disconnect");
        client.emit("auth", { token: token("bob") });
        assert.equal(await disconnected, "io server disconnect");
    });

    it("drops a socket that renews with someone else's token", async () => {
        const { client } = await member("bob", "viewer", 1);
        await once(client, "token_expiring");

        const ana = token("ana");
        roles.set(ana, "owner");
        const disconnected = once<string>(client, "disconnect");
        client.emit("auth", { token: ana });
        assert.equal(await disconnected, "io server disconnect");
    });
});
