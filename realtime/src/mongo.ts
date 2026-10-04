import { MongoClient, type ChangeStream, type ResumeToken } from "mongodb";

const mongo = new MongoClient(process.env.MONGODB_URI!, {
    appName: "tesis.vercel.integration",
});

const docs = await mongo.connect();

const boards = docs
    .db(process.env.MONGO_DATABASE || "prod")
    .collection("boards");

export type Callback = (id: string, board: unknown) => void;

let callback: Callback = () => {};
export const setStreamCallback = (cb: Callback) => (callback = cb);

const REOPEN_DELAY_MS = 1000;
// The oplog no longer holds the change the token points at.
const HISTORY_LOST = 286;

let stream: ChangeStream;
let resumeToken: ResumeToken | undefined;
let closing = false;

// The driver resumes the stream across brief failures by itself; one that
// gives up (Mongo down for longer) is reopened from the last change seen.
function watch() {
    stream = boards.watch([{ $match: { operationType: "update" } }], {
        fullDocument: "updateLookup",
        ...(resumeToken ? { resumeAfter: resumeToken } : {}),
    });

    stream
        .on("change", (event) => {
            resumeToken = event._id;
            const change = event as typeof event & { operationType: "update" };

            const board = change.fullDocument;
            const id = change.documentKey._id;

            if (board) callback(id.toString(), board);
        })
        .once("error", (err: Error & { code?: unknown }) => {
            console.error("Board change stream failed:", err.message);
            if (err.code === HISTORY_LOST) resumeToken = undefined;
            if (!closing) setTimeout(watch, REOPEN_DELAY_MS);
        });
}

watch();

export async function close() {
    closing = true;
    await stream.close();
    return docs.close();
}
