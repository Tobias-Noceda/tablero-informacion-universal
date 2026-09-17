import * as cache from "./redis.js";

import { Server } from "socket.io";

import { createServer } from "node:http";

export const server = createServer();

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

wss.on("connection", async (socket) => {
    if (socket.recovered) {
        return;
    }

    const { peer, board } = socket.handshake.query;

    if (
        typeof peer !== "string" ||
        typeof board !== "string" ||
        board.trim() === "" ||
        peer.trim() === ""
    ) {
        socket.disconnect(true);
        return;
    }

    try {
        const peers = await cache.getAndInsert(board, peer);
        socket.emit("peers", peers);
    } catch (e) {
        console.error("Failed to register peer", e);
        socket.disconnect(true);
        return;
    }

    socket.join(board);

    socket.on("disconnect", (reason) => {
        console.error("Client disconnected:", reason);
        cache.remove(board, peer);
    });
});
