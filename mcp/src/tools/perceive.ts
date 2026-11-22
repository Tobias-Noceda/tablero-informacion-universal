import type { BoardPostIt, PostItData } from "../types/api";

import type { ToolMetadata, InferSchema } from "xmcp";

import * as api from "../modules/api";
import {
    credentialsOf,
    describeData,
    liveCardBody,
    MAX_REQUEST_BODY,
    requestBody,
} from "../modules/live";

import { z } from "zod";

const fields = z
    .record(z.string().trim().min(1), z.string())
    .refine((map) => Object.keys(map).length <= 20, "At most 20 entries");

export const schema = {
    board_id: z.uuid().describe("The memory PKG ID"),
    title: z
        .string()
        .trim()
        .min(1)
        .max(128)
        .describe(
            "The memory title. It may show the values it reads as {{name}}, e.g. \"Bitcoin: {{price}} USD\"",
        ),
    url: z
        .url({ protocol: /^https?$/ })
        .describe("The http(s) address the memory reads, without its query string"),
    method: z
        .enum(["GET", "POST"])
        .default("GET")
        .describe("GET, or POST for APIs that take their search in a JSON body"),
    body: z
        .unknown()
        .optional()
        .describe(
            "Only with POST: the JSON body (an object or an array), sent as application/json, at most 16 KB. A string value written as $NAME uses the board credential stored under NAME",
        ),
    response: z
        .enum(["json", "html"])
        .default("json")
        .describe("What the address answers: json (read with jq) or html (read with CSS selectors)"),
    query: fields
        .refine((map) => Object.keys(map).length > 0, "At least one value to read")
        .describe(
            "The values to keep, by name: a jq expression for json (\".bitcoin.usd\", \".items[0].title\") or a CSS selector for html (\"h1\", \".price\")",
        ),
    queries: fields
        .optional()
        .describe("Query string parameters, by name. A value written as $NAME uses the board credential stored under NAME"),
    headers: fields
        .optional()
        .describe("Request headers, by name. A value written as $NAME uses the board credential stored under NAME"),
    rate: z
        .int()
        .min(6)
        .max(86400)
        .default(300)
        .describe("How often, in seconds, the memory reads the address again (more than 5)"),
    relevant: z.array(z.uuid()).default([]).describe("Related memories to this memory"),
    position: z
        .object({
            x: z.number().describe("X coordinate"),
            y: z.number().describe("Y coordinate"),
        })
        .describe("The memory position"),
};

export const metadata: ToolMetadata = {
    name: "perceive",
    description:
        "This tool lets you save a live memory: instead of a fixed text, it reads an external API (or web page), with a GET or a POST carrying a JSON body, and keeps the values you pick, refreshing them every `rate` seconds, so retrieving it later always shows current data. Use it for things that change, such as prices, weather, statuses or counters. The memory reads the address once right away and you get what it read, so you can check your jq expressions or CSS selectors; if the read fails, the memory is not kept and you get the reason, so fix the request and try again. Never put a secret (an API key, a token) in a header or query value: write $NAME instead and ask the user to store that credential, under that name, in the board's credentials panel. You must provide a spacial position, in pixels, for the memory in the board, so make an educated guess.",
    annotations: {
        title: "Perceive a live memory",
        readOnlyHint: false,
        destructiveHint: false,
        idempotentHint: false,
        openWorldHint: true,
    },
};

export default async function ({
    board_id,
    title,
    url,
    method,
    body,
    response,
    query,
    queries,
    headers,
    rate,
    relevant,
    position,
}: InferSchema<typeof schema>) {
    if (method === "GET" && body !== undefined) {
        return "A GET request carries no body: use method POST to send one.";
    }
    const memory = { board: board_id, title, url, method, body, response, query, queries, headers, rate };
    const size = new TextEncoder().encode(requestBody(memory)).length;
    if (size > MAX_REQUEST_BODY) {
        return `The body is ${size} bytes; a memory's body may be at most ${MAX_REQUEST_BODY}.`;
    }

    const res = await api.post("/v1/post-its", liveCardBody(memory));
    const postit = (await res.json()) as BoardPostIt;

    // A memory that can't read its address is worth nothing on the board.
    let data: PostItData;
    try {
        data = (await (await api.get(`/v1/post-its/${postit.id}`)).json()) as PostItData;
    } catch (err) {
        await api.del(`/v1/post-its/${postit.id}`).catch(() => undefined);
        const reason = err instanceof Error ? err.message : String(err);
        const needed = credentialsOf(memory);
        const hint = needed.length
            ? ` It names these board credentials, which must exist: ${needed.join(", ")}.`
            : "";
        return `The memory could not read ${url}, so it was not kept: ${reason}${hint}`;
    }

    await api.patch(`/v1/post-its/${postit.id}/position`, position);
    await Promise.all(
        relevant.map((target) =>
            api.post(`/v1/boards/${board_id}/strands`, { source: postit.id, target }),
        ),
    );

    return `Created live memory ${postit.id}, refreshed every ${rate} s. It read:\n${describeData(data)}`;
}
