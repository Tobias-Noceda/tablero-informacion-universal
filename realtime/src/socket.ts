import * as cache from "./redis.js";

import { Server } from "socket.io";

import { createServer } from "node:http";

export const server = createServer();

const io = new Server(server, {
    path: "/ws",
});

export function notify(group: string, data: unknown) {
    io.to(group).emit("update", {
        data,
        ts: Date.now(),
    });
}

export function close() {
    return Promise.allSettled([io.close(), cache.close()]);
}

io.on("connection", async (socket) => {
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
