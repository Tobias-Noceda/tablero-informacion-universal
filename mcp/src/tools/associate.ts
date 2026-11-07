import type { ToolMetadata, InferSchema } from "xmcp";

import * as api from "../modules/api";

import { z } from "zod";

// Define the schema for tool parameters
export const schema = {
    board_id: z.uuid().describe("The memory PKG ID"),
    id: z.uuid().describe("The memory ID"),
    relevant: z.array(z.uuid()).describe("Related memories to this memory"),
};

// Define tool metadata
export const metadata: ToolMetadata = {
    name: "associate",
    description:
        "This tool lets you associate relevant memories together. You should use this tool infrequently, as the memories should only be connected on creation and rarely does a new relation appear without impliying a new third memory appeared.",
    annotations: {
        title: "Associate memories",
        readOnlyHint: false,
        destructiveHint: false,
        idempotentHint: true,
    },
};

// Tool implementation
export default async function ({
    board_id,
    id,
    relevant,
}: InferSchema<typeof schema>) {
    const promises = relevant.map((target) =>
        api.post(`/v1/boards/${board_id}/strands`, {
            source: id,
            target,
        }),
    );

    await Promise.all(promises);

    return `Associated ${id} with ${relevant.join(", ")}`;
}
