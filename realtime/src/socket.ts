import type { Server as HttpServer } from "node:http";

import { Server, type Socket } from "socket.io";

import { claimsOf, type BoardRole, type RoleLookup } from "./auth.js";

export interface Presence {
    // Records the peer and answers the ones already there (peer → user).
    join(
        board: string,
        peer: string,
        user: string,
    ): Promise<Record<string, string>>;
    leave(board: string, peer: string): Promise<void>;
}

export type Options = {
    roleOn: RoleLookup;
    presence: Presence;
    // How long before the token expires the client is asked for a new one.
    renewLead?: number;
    // How long after it expires a socket without a new one is dropped.
    grace?: number;
};

export type Realtime = {
    notify(board: string, doc: unknown): void;
    close(): Promise<void>;
};

type SocketData = {
    user: string;
    board: string;
    peer: string;
    role: BoardRole;
    exp: number;
};

type ClientEvents = {
    auth: (payload: unknown) => void;
};

type ServerEvents = {
    peers: (peers: Record<string, string>) => void;
    update: (update: { board: unknown; ts: number }) => void;
    token_expiring: () => void;
};

type BoardServer = Server<
    ClientEvents,
    ServerEvents,
    Record<string, never>,
    SocketData
>;
type BoardSocket = Socket<
    ClientEvents,
    ServerEvents,
    Record<string, never>,
    SocketData
>;

const uuid =
    /^[0-9A-F]{8}-[0-9A-F]{4}-[1-5][0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12}$/i;

function validUUID(s: unknown): s is string {
    return typeof s === "string" && uuid.test(s);
}

function tokenOf(payload: unknown): string | null {
    const token = (payload as { token?: unknown } | null)?.token;
    return typeof token === "string" && token !== "" ? token : null;
}

// Board rooms for whoever the platform API lets see the board. A socket that
// does not renew its token, or loses the board, is dropped.
export function attach(
    server: HttpServer,
    { roleOn, presence, renewLead = 60_000, grace = 30_000 }: Options,
): Realtime {
    const wss: BoardServer = new Server(server, { path: "/ws" });

    wss.use(async (socket, next) => {
        const { board, peer } = socket.handshake.query;
        if (!validUUID(board) || !validUUID(peer)) {
            return next(new Error("invalid_handshake"));
        }

        const token = tokenOf(socket.handshake.auth);
        const claims = token && claimsOf(token);
        const role = claims && (await roleOn(token, board));
        if (!claims || !role) return next(new Error("unauthorized"));

        socket.data = {
            user: claims.sub,
            board: board.toLowerCase(),
            peer: peer.toLowerCase(),
            role,
            exp: claims.exp,
        };
        next();
    });

    wss.on("connection", async (socket: BoardSocket) => {
        const { board, peer, user } = socket.data;
        let expiring: NodeJS.Timeout | undefined;
        let deadline: NodeJS.Timeout | undefined;

        const schedule = () => {
            clearTimeout(expiring);
            clearTimeout(deadline);
            const left = socket.data.exp * 1000 - Date.now();
            expiring = setTimeout(
                () => socket.emit("token_expiring"),
                Math.max(0, left - renewLead),
            );
            deadline = setTimeout(
                () => socket.disconnect(true),
                Math.max(0, left + grace),
            );
        };

        socket.on("disconnect", () => {
            clearTimeout(expiring);
            clearTimeout(deadline);
            presence.leave(board, peer).catch((err) => {
                console.error("Failed to forget peer", err);
            });
        });

        socket.on("auth", async (payload) => {
            const token = tokenOf(payload);
            const claims = token && claimsOf(token);
            const role =
                claims && claims.sub === user && (await roleOn(token, board));
            if (!claims || !role) {
                socket.disconnect(true);
                return;
            }

            socket.data.role = role;
            socket.data.exp = claims.exp;
            schedule();
            // Keeps the presence entry from expiring.
            presence.join(board, peer, user).catch((err) => {
                console.error("Failed to refresh peer", err);
            });
        });

        let peers: Record<string, string>;
        try {
            peers = await presence.join(board, peer, user);
        } catch (err) {
            console.error("Failed to register peer", err);
            socket.disconnect(true);
            return;
        }
        // Gone while joining: its disconnect ran before the peer was recorded.
        if (socket.disconnected) {
            await presence.leave(board, peer).catch(() => {});
            return;
        }

        socket.emit("peers", peers);

        socket.join(board);
        schedule();
    });

    return {
        notify(board, doc) {
            wss.to(board.toLowerCase()).emit("update", {
                board: doc,
                ts: Date.now(),
            });
        },
        close() {
            return wss.close();
        },
    };
}
