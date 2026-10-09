import { createServer } from "node:http";

import { boardRoles } from "./auth.js";
import * as mongo from "./mongo.js";
import * as presence from "./redis.js";
import { attach } from "./socket.js";

const api = process.env.API_URL;
if (!api)
    throw new Error("API_URL is required, e.g. http://backend:31126/api/v1");

const server = createServer();
const realtime = attach(server, { roleOn: boardRoles(api), presence });

mongo.setStreamCallback(realtime.notify);

if (import.meta.main) {
    const port = process.env.PORT ?? 3000;
    server.listen(port);
}

process.on("SIGTERM", () =>
    Promise.allSettled([realtime.close(), presence.close(), mongo.close()]),
);

export default server;
