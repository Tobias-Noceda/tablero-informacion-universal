import type { ToolMetadata, InferSchema } from "xmcp";

import * as api from "../modules/api";
import { describeResponse } from "../modules/card-types";
import { MAX_REQUEST_BODY, requestBody } from "../modules/live";

import { z } from "zod";

const fields = z
    .record(z.string().trim().min(1), z.string())
    .refine((map) => Object.keys(map).length <= 20, "At most 20 entries");

export const schema = {
    board_id: z
        .uuid()
        .describe(
            "The board (PKG) to try it from: its credentials are the ones $NAME uses",
        ),
    url: z
        .url({ protocol: /^https?$/ })
        .describe("The http(s) address, without its query string"),
    method: z
        .enum(["GET", "POST"])
        .default("GET")
        .describe("GET, or POST with a JSON body"),
    body: z
        .unknown()
        .optional()
        .describe("Only with POST: the JSON body, at most 16 KB"),
    queries: fields
        .optional()
        .describe(
            "Query string parameters, by name. A value written as $NAME uses the board credential stored under NAME",
        ),
    headers: fields
        .optional()
        .describe(
            "Request headers, by name. A value written as $NAME uses the board credential stored under NAME",
        ),
};

export const metadata: ToolMetadata = {
    name: "try_request",
    description:
        "This tool makes a request once, as a card would, and gives you its whole JSON answer without saving anything, so you can write the jq expressions of a live memory (perceive) or a card type (create_card_type) against what the API really returns. Long answers are cut. Requests are limited to a few per minute.",
    annotations: {
        title: "Try a request",
        readOnlyHint: true,
        destructiveHint: false,
        idempotentHint: false,
        openWorldHint: true,
    },
};

export default async function ({
    board_id,
    url,
    method,
    body,
    queries,
    headers,
}: InferSchema<typeof schema>) {
    if (method === "GET" && body !== undefined) {
        return "A GET request carries no body: use method POST to send one.";
    }
    const text = requestBody({ method, body });
    if (new TextEncoder().encode(text).length > MAX_REQUEST_BODY) {
        return `The body may be at most ${MAX_REQUEST_BODY} bytes.`;
    }

    try {
        const res = await api.post("/v1/post-its/preview", {
            board: board_id,
            resource: url,
            request: {
                method,
                headers: headers ?? {},
                queries: queries ?? {},
                ...(text && { body: text }),
            },
            response: "json",
        });
        const answer = (await res.json()) as { response?: unknown };
        return `${url} answered:\n${describeResponse(answer.response)}`;
    } catch (err) {
        if (err instanceof api.ApiError && err.status === 429) {
            return "Too many tries in a row: wait a minute before trying again.";
        }
        if (err instanceof api.ApiError)
            return `The request failed: ${err.message}`;
        throw err;
    }
}
