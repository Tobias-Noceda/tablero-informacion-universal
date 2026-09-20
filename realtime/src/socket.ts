import * as cache from "./redis.js";

import { Server } from "socket.io";

import { createServer } from "node:http";

export const server = createServer();

type UUID = string;

const uuid =
    /^[0-9A-F]{8}-[0-9A-F]{4}-[1-5][0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12}$/i;

const wss = new Server(server, {
    path: "/ws",
});

export function notify(group: string, board: unknown) {
    wss.to(group).emit("update", {
        board,
        ts: Date.now(),
    });
}

export function close() {
    return Promise.allSettled([wss.close(), cache.close()]);
}

function validUUID(s: unknown): s is UUID {
    return typeof s === "string" && uuid.test(s);
}

wss.on("connection", async (socket) => {
    if (socket.recovered) {
        return;
    }

    const { board, user, peer } = socket.handshake.query;

    if (!validUUID(board) || !validUUID(user) || !validUUID(peer)) {
        socket.disconnect(true);
        return;
    }

    try {
        const peers = await cache.getAndInsert(board, user, peer);
        socket.emit("peers", peers);
    } catch (e) {
        console.error("Failed to register peer", e);
        socket.disconnect(true);
        return;
    }

    socket.join(board);

    socket.on("disconnect", (reason) => {
        console.error("Client disconnected:", reason);
        cache.remove(board, user);
    });
});
