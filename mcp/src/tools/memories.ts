import type { Board } from "../types/api";

import type { ToolMetadata, InferSchema } from "xmcp";

import * as api from "../modules/api";

import { z } from "zod";

// Define the schema for tool parameters
export const schema = {
    id: z.uuid().describe("The memory PKG ID"),
};

// Define tool metadata
export const metadata: ToolMetadata = {
    name: "memories",
    description:
        "This tool provides you all the memories stored in a specific personal knowledge graph, or PKG. Each memory is stored in 'cards' as `id - position (x; y) - memory`, which may take variables that are automatically populated from different sources (such as APIs) on retrieval. A list of related memories is provided when retrieving a specific memory, in order to further search for more details on a specific topic. This tool should be called infrequently, only when doing a full search from scratch or the context is empty. It is expected from you to have a 'working memory', in which you remember a few things related to your task at hand, in particular the tuple `id - memory` of what you consider most important to complete your job.",
    annotations: {
        title: "Retrieve all the memories from a personal knowledge graph",
        readOnlyHint: true,
        destructiveHint: false,
        idempotentHint: true,
    },
};

// Tool implementation
export default async function ({ id }: InferSchema<typeof schema>) {
    const res = await api.get(`/v1/boards/${id}`);
    const board = (await res.json()) as Board;

    const memories = board.postits.map(
        ({ id, position, title }) =>
            `${id} - (${position.x}; ${position.y}) - ${title.text}`,
    );

    return memories.join("\n");
}
