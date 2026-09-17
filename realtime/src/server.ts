import * as socket from "./socket.js";
import * as mongo from "./mongo.js";

if (import.meta.main) {
    mongo.setStreamCallback(socket.notify);

    const port = process.env.PORT ?? 3000;
    socket.server.listen(port);

    process.on("SIGTERM", () =>
        Promise.allSettled([socket.close(), mongo.close()]),
    );
}

export default socket.server;
