import type { BoardPostIt } from "../types/api";

import type { ToolMetadata, InferSchema } from "xmcp";

import * as api from "../modules/api";

import { z } from "zod";

// Define the schema for tool parameters
export const schema = {
    board_id: z.uuid().describe("The memory PKG ID"),
    title: z.string().trim().length(128).describe("The memory title"),
    description: z.string().trim().describe("The memory description"),
    relevant: z.array(z.uuid()).describe("Related memories to this memory"),
    position: z
        .object({
            x: z.number().describe("X coordinate"),
            y: z.number().describe("Y coordinate"),
        })
        .describe("The memory position"),
};

// Define tool metadata
export const metadata: ToolMetadata = {
    name: "consolidate",
    description:
        "This tool lets you save a text memory, with connections to other relevant memories. You should preserve anything important for your task to be able to retrieve it later. Each memory should be concise, detailed and specific. Bigger concepts should be splited in smaller interconected memories. You must provide a spacial position, in pixels, for the memory in the board, so make an educated guess. Try to save similar memories close to each other, but positioning is purely visual for the user.",
    annotations: {
        title: "Consolidate a memory",
        readOnlyHint: false,
        destructiveHint: false,
        idempotentHint: false,
    },
};

// Tool implementation
export default async function ({
    board_id,
    title,
    description,
    relevant,
    position,
}: InferSchema<typeof schema>) {
    const res = await api.post("/v1/post-its", {
        board: board_id,
        title: { text: title, vars: false },
        well_known: "static_card",
        params: {
            text: description,
        },
    });

    const postit = (await res.json()) as BoardPostIt;

    await api.patch(`/v1/post-its/${postit.id}/position`, position);

    const promises = relevant.map((target) =>
        api.post(`/v1/boards/${board_id}/strands`, {
            source: postit.id,
            target,
        }),
    );

    await Promise.all(promises);

    return `Created memory ${postit.id}`;
}
