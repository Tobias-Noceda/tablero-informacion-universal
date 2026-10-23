import type { Board, PostItData } from "../types/api";

import type { ToolMetadata, InferSchema } from "xmcp";

import * as api from "../modules/api";

import { z } from "zod";
import Mustache from "mustache";

// Define the schema for tool parameters
export const schema = {
    board_id: z.uuid().describe("The memory table ID"),
    postit_id: z.uuid().describe("The memory ID"),
};

// Define tool metadata
export const metadata: ToolMetadata = {
    name: "retrieve",
    description:
        "Retrieve a memory: its title, its data and the memories connected to it",
    annotations: {
        title: "Retrieve a memory",
        readOnlyHint: true,
        destructiveHint: false,
        idempotentHint: false,
    },
};

// Tool implementation
export default async function ({
    board_id,
    postit_id,
}: InferSchema<typeof schema>) {
    const [b_res, d_res] = await Promise.all([
        api.get(`/v1/boards/${board_id}`),
        api.get(`/v1/post-its/${postit_id}`),
    ]);

    const board = (await b_res.json()) as Board;
    const data = (await d_res.json()) as PostItData;

    const postit = board.postits.find((p) => p.id === postit_id);

    if (!postit) throw new Error("That memory is not on this board.");

    const title = postit.title.vars
        ? Mustache.render(postit.title.text, data)
        : postit.title.text;

    const related_ids = new Set(
        board.strands
            .map((p) =>
                p.source === postit_id
                    ? p.target
                    : p.target === postit_id
                      ? p.source
                      : null,
            )
            .filter((id) => id !== null),
    );

    const related = board.postits
        .filter((p) => related_ids.has(p.id))
        .map(({ id, title }) => `${id} - ${title.text}`);

    return [
        title,
        "",
        "Data:",
        JSON.stringify(data, null, 2),
        "",
        "Related memories:",
        related.join("\n") || "(none)",
    ].join("\n");
}
