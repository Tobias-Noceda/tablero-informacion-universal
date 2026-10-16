import { createClient } from "redis";

const redis = createClient({ url: process.env.REDIS_URL! });
// The client reconnects on its own; without a listener the error would
// end the process.
redis.on("error", (err: Error) => console.error("Redis:", err.message));
const cache = await redis.connect();

// Also forgets peers that vanished without disconnecting.
const ONLINE_TTL_SECONDS = 20 * 60;

// A hash of peer → user: each tab is its own entry.
function onlineKey(board: string) {
    return `board:${board.toLowerCase()}:online`;
}

export async function join(board: string, peer: string, user: string) {
    const key = onlineKey(board);

    const [peers] = await cache
        .multi()
        .hGetAll(key)
        .hSet(key, peer.toLowerCase(), user)
        .expire(key, ONLINE_TTL_SECONDS)
        .exec();

    return peers as unknown as Record<string, string>;
}

export async function leave(board: string, peer: string) {
    await cache.hDel(onlineKey(board), peer.toLowerCase());
}

export async function close() {
    await cache.close();
}
