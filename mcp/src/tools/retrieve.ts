import type { Board } from "../types/api";

import type { ToolMetadata, InferSchema } from "xmcp";

import * as api from "../modules/api";
import { dataOf, lines, titleOf } from "../modules/memory";

import { z } from "zod";

// Define the schema for tool parameters
export const schema = {
    board_id: z.uuid().describe("The memory PKG ID"),
    postit_id: z.uuid().describe("The memory ID"),
};

// Define tool metadata
export const metadata: ToolMetadata = {
    name: "retrieve",
    description:
        "This tool lets you retrieve a specific memory, with all the details and variables populated. At retrieval, related memories will be reactivated, in order to help remembering other important details that may be required. Memories may be refreshed from time to time, as to always have the updated details, in particular for memories with variables that may change frequently.",
    annotations: {
        title: "Retrieve a memory",
        readOnlyHint: true,
        destructiveHint: false,
        idempotentHint: true,
        openWorldHint: true,
    },
};

// Tool implementation
export default async function ({
    board_id,
    postit_id,
}: InferSchema<typeof schema>) {
    const b_res = await api.get(`/v1/boards/${board_id}`);
    const board = (await b_res.json()) as Board;

    const postit = board.postits.find((p) => p.id === postit_id);

    if (!postit) throw new Error("That memory is not on this board.");

    const data = await dataOf(postit_id);
    const title = await titleOf(postit, data);

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

    const related = await lines(
        board.postits.filter((p) => related_ids.has(p.id)),
    );

    return `${JSON.stringify(title)}

Data: ${JSON.stringify(data)}

Related memories:
${related.join("\n") || "(none)"}
`;
}
