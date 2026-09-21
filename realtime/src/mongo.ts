import { MongoClient } from "mongodb";

const mongo = new MongoClient(process.env.MONGODB_URI!, {
    appName: "tesis.vercel.integration",
});

const docs = await mongo.connect();

const boards = docs
    .db(process.env.MONGO_DATABASE || "prod")
    .collection("boards");

const stream = boards.watch([{ $match: { operationType: "update" } }], {
    fullDocument: "updateLookup",
});

export type Callback = (id: string, board: unknown) => void;

let callback: Callback = () => {};
export const setStreamCallback = (cb: Callback) => (callback = cb);

export async function close() {
    await stream.close();
    return docs.close();
}

stream
    .on("change", (event) => {
        const change = event as typeof event & { operationType: "update" };

        const board = change.fullDocument;
        const id = change.documentKey._id;

        if (board) callback(id.toString(), board);
    })
    .once("error", console.error);
