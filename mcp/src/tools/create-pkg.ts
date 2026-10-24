import type { Board } from "../types/api";

import type { ToolMetadata, InferSchema } from "xmcp";

import * as api from "../modules/api";

import { z } from "zod";

// Define the schema for tool parameters
export const schema = {
    name: z.string().trim().length(32).describe("The memory PKG name"),
};

// Define tool metadata
export const metadata: ToolMetadata = {
    name: "PKGs",
    description:
        "This tool provides you a way to create a personal knowledge graph, or PKG. Each PKG is a collection of memories you can organize in a canvas, joining related topics to one another, and to later search, update or delete the memories as deemed necesary.",
    annotations: {
        title: "Create a personal knowledge graphs",
        readOnlyHint: false,
        destructiveHint: false,
        idempotentHint: false,
    },
};

// Tool implementation
export default async function ({ name }: InferSchema<typeof schema>) {
    const res = await api.post(`/v1/boards/`, { name });
    const board = (await res.json()) as Board;
    return `Created board ${board.name} (${board.id})`;
}
