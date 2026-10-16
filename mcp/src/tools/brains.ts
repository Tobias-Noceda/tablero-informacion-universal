import type { Board } from "../types/api";

import type { ToolMetadata, InferSchema } from "xmcp";

import * as api from "../modules/api";

import { z } from "zod";

// Define the schema for tool parameters
export const schema = {};

// Define tool metadata
export const metadata: ToolMetadata = {
    name: "brains",
    description: "Retrieve all the memory boards",
    annotations: {
        title: "Retrieve all the memory boards",
        readOnlyHint: true,
        destructiveHint: false,
        idempotentHint: false,
    },
};

// Tool implementation
export default async function () {
    const res = await api.get(`/v1/boards`);
    const postits = (await res.json()) as Board[];

    const memories = postits.map(({ id, name }) => `${id} - ${name}`);
    return memories.join("\n");
}
