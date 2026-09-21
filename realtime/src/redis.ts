import { createClient } from "redis";

const redis = createClient({ url: process.env.REDIS_URL! });
const cache = await redis.connect();

function onlineKey(board: string) {
    return `board:${board.toLowerCase()}:online`;
}

export async function getAndInsert(board: string, user: string, peer: string) {
    const key = onlineKey(board);

    const [peers] = await cache
        .multi()
        .hGetAll(key)
        .hSetEx(
            key,
            { [user.toLowerCase()]: peer.toLowerCase() },
            { expiration: { type: "EX", value: 20 * 60 } },
        )
        .exec();

    return peers as unknown as Record<string, string>;
}

export async function remove(board: string, user: string) {
    const key = onlineKey(board);
    await cache.hDel(key, user.toLowerCase());
}

export function close() {
    return cache.close();
}
