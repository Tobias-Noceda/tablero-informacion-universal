import type { Board } from "../types/api";

import type { ToolMetadata, InferSchema } from "xmcp";

import * as api from "../modules/api";
import { dataOf, lines, titleOf } from "../modules/memory";
import { relation } from "../modules/strands";

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
        "This tool lets you retrieve a specific memory, with all the details and variables populated. At retrieval, related memories will be reactivated, in order to help remembering other important details that may be required. Each related memory starts with how it relates to this one: → this memory points to it, ← it points to this memory, ↔ both ways, — undirected, followed by the relation's label when it has one. Memories may be refreshed from time to time, as to always have the updated details, in particular for memories with variables that may change frequently.",
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

    // The other end of every strand touching this memory, with how it reads.
    const relations = new Map<string, string>();
    for (const strand of board.strands) {
        const other =
            strand.source === postit_id
                ? strand.target
                : strand.target === postit_id
                  ? strand.source
                  : null;
        if (other) relations.set(other, relation(strand, postit_id));
    }

    const related_postits = board.postits.filter((p) => relations.has(p.id));
    const related = (await lines(related_postits)).map(
        (line, i) => `${relations.get(related_postits[i].id)} ${line}`,
    );

    return `${JSON.stringify(title)}

Data: ${JSON.stringify(data)}

Related memories:
${related.join("\n") || "(none)"}
`;
}
