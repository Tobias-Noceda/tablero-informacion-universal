import { createClient } from "redis";

export default createClient({ url: process.env.REDIS_URL! });
