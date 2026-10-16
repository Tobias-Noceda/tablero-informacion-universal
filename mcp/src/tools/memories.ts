import type { Board } from "../types/api";

import type { ToolMetadata, InferSchema } from "xmcp";

import * as api from "../modules/api";

import { z } from "zod";

// Define the schema for tool parameters
export const schema = {
    id: z.uuid().describe("The memory table ID"),
};

// Define tool metadata
export const metadata: ToolMetadata = {
    name: "memories",
    description: "Retrieve all the agents memories",
    annotations: {
        title: "Retrieve all the agents memories",
        readOnlyHint: true,
        destructiveHint: false,
        idempotentHint: false,
    },
};

// Tool implementation
export default async function ({ id }: InferSchema<typeof schema>) {
    const res = await api.get(`/v1/boards/${id}`);
    const board = (await res.json()) as Board;

    const memories = board.postits.map(
        ({ id, title }) => `${id} - ${title.text}`,
    );

    return memories.join("\n");
}
