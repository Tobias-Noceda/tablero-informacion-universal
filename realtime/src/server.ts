import * as socket from "./socket.js";

import mongo from "./mongo.js";

const docs = await mongo.connect();

const boards = docs.db("prod").collection("boards");
const stream = boards.watch([{ $match: { operationType: "update" } }], {
    fullDocument: "updateLookup",
});

stream
    .on("change", (event) => {
        const change = event as typeof event & { operationType: "update" };

        const board = change.fullDocument;
        const id = change.documentKey._id;

        if (!board) return;

        socket.notify(id.toString(), board);
    })
    .once("error", console.error);

if (import.meta.main) {
    const port = process.env.PORT ?? 3000;
    socket.server.listen(port);
}

process.on("SIGTERM", async () => {
    await stream.close();
    await Promise.allSettled([socket, docs].map((r) => r.close()));
});

export default socket.server;
