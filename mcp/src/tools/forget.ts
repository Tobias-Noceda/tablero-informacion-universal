import type { ToolMetadata, InferSchema } from "xmcp";

import * as api from "../modules/api";

import { z } from "zod";

// Define the schema for tool parameters
export const schema = {
    id: z.uuid().describe("The memory ID"),
};

// Define tool metadata
export const metadata: ToolMetadata = {
    name: "forget",
    description:
        "This tool provides you a way to forget old, unnecesary or outdated memories. Irrelevant memories that don't add to your knowledge will impact on your performance to actually remebering important things, hence why it's reasonable to purge old details from time to time. However, don't over do it, as sometimes it's good to remember for some extra time some other details.",
    annotations: {
        title: "Forget a memory from a personal knowledge graph",
        readOnlyHint: false,
        destructiveHint: true,
        idempotentHint: true,
    },
};

// Tool implementation
export default async function ({ id }: InferSchema<typeof schema>) {
    await api.del(`/v1/post-its/${id}`);
    return "Memory deleted";
}
