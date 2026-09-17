import mongo from "./mongo.js";
import * as cache from "./redis.js";

import { Server } from "socket.io";

import { createServer } from "node:http";

const server = createServer();

const io = new Server(server, {
    path: "/ws",
    connectionStateRecovery: {
        maxDisconnectionDuration: 2 * 60 * 1000,
    },
});

const docs = await mongo.connect();

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
        const peers = cache.getAndInsert(board, peer);
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

const boards = docs.db("prod").collection<{ _id: string }>("boards");
const stream = boards.watch([{ $match: { operationType: "update" } }], {
    fullDocument: "updateLookup",
});

stream
    .on("change", (event) => {
        const change = event as typeof event & { operationType: "update" };

        const board = change.fullDocument;
        const id = change.documentKey._id;

        if (!board) return;

        io.to(id.toString()).emit("update", {
            board,
            ts: Date.now(),
        });
    })
    .once("error", console.error);

if (import.meta.main) {
    const port = process.env.PORT ?? 3000;
    server.listen(port);
}

process.on("SIGTERM", async () => {
    await stream.close();
    await Promise.allSettled([io, cache, docs].map((r) => r.close()));
});

export default server;
