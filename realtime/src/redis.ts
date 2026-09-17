import { createClient } from "redis";

const redis = createClient({ url: process.env.REDIS_URL! });
const cache = await redis.connect();

function onlineKey(board: string) {
    return `board:${board}:online`;
}

export async function getAndInsert(
    board: string,
    peer: string,
): Promise<unknown> {
    const key = onlineKey(board);
    const [peers] = await cache.multi().sMembers(key).sAdd(key, peer).exec();
    return peers;
}

export async function remove(board: string, peer: string) {
    const key = onlineKey(board);
    cache.sRem(key, peer);
}

export function close() {
    return cache.close();
}
