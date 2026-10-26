import type { ToolMetadata, InferSchema } from "xmcp";

import * as api from "../modules/api";

import { z } from "zod";

// Define the schema for tool parameters
export const schema = {
    id: z.uuid().describe("The memory PKG ID"),
};

// Define tool metadata
export const metadata: ToolMetadata = {
    name: "PKGs",
    description:
        "This tool provides you a way to delete personal knowledge graph, or PKG.",
    annotations: {
        title: "Delete a personal knowledge graphs you have access to",
        readOnlyHint: false,
        destructiveHint: true,
        idempotentHint: true,
    },
};

// Tool implementation
export default async function ({ id }: InferSchema<typeof schema>) {
    await api.del(`/v1/boards/${id}`);
    return "PKG deleted";
}
