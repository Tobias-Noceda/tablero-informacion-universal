import type { Board } from "../types/api";

import type { ToolMetadata, InferSchema } from "xmcp";

import * as api from "../modules/api";

import { z } from "zod";

// Define the schema for tool parameters
export const schema = {
    board_id: z.uuid().describe("The memory PKG ID"),
    memory_id_1: z.uuid().describe("One memory ID"),
    memory_id_2: z.uuid().describe("Other memory ID"),
};

// Define tool metadata
export const metadata: ToolMetadata = {
    name: "dissociate",
    description:
        "This tool lets you dissociate memories from each other. This tool is counter-productive to your PKG based memory, as you have no reason to just forget something, and your strongest feature is being able to retain details forever. However, the option is here for completition.",
    annotations: {
        title: "Dissociate two memories",
        readOnlyHint: false,
        destructiveHint: true,
        idempotentHint: true,
    },
};

// Tool implementation
export default async function ({
    board_id,
    memory_id_1,
    memory_id_2,
}: InferSchema<typeof schema>) {
    const res = await api.get(`/v1/boards/${board_id}`);
    const board = (await res.json()) as Board;

    const promises = board.strands
        .filter(
            ({ source, target }) =>
                (source === memory_id_1 && target === memory_id_2) ||
                (source === memory_id_2 && target === memory_id_1),
        )
        .map((s) => api.del(`/v1/boards/${board_id}/strands/${s.id}`));

    await Promise.all(promises);

    return `Dissociated ${memory_id_1} from ${memory_id_2}`;
}
