import type { Board, Strand } from "../types/api";

import type { ToolMetadata, InferSchema } from "xmcp";

import * as api from "../modules/api";
import { arrowFrom, strandBetween } from "../modules/strands";

import { z } from "zod";

// Define the schema for tool parameters
export const schema = {
    board_id: z.uuid().describe("The memory PKG ID"),
    id: z.uuid().describe("The memory ID"),
    relevant: z.array(z.uuid()).describe("Related memories to this memory"),
    label: z
        .string()
        .trim()
        .max(80)
        .optional()
        .describe(
            'What relates them, read from this memory to the related ones (e.g. "depends on", "is part of"). Applies to every related memory. Leave unset to keep existing labels; an empty string removes them.',
        ),
    directed: z
        .boolean()
        .optional()
        .describe(
            "Whether the relation points from this memory to the related ones (drawn as an arrow). Leave unset to keep the direction of existing associations; new ones are then undirected.",
        ),
};

// Define tool metadata
export const metadata: ToolMetadata = {
    name: "associate",
    description:
        "This tool lets you associate relevant memories together, optionally naming the relation and its direction. Associating memories that are already associated updates the relation instead of duplicating it. You should use this tool infrequently, as the memories should only be connected on creation and rarely does a new relation appear without impliying a new third memory appeared.",
    annotations: {
        title: "Associate memories",
        readOnlyHint: false,
        destructiveHint: false,
        idempotentHint: true,
    },
};

type Style = Pick<Strand, "label" | "arrow">;

// What an existing strand must change to match the request, if anything.
function restyle(
    strand: Strand,
    from: string,
    label?: string,
    directed?: boolean,
) {
    const patch: Style = {};
    if (label !== undefined) patch.label = label;
    if (directed !== undefined)
        patch.arrow = directed ? arrowFrom(strand, from) : "none";
    return Object.keys(patch).length > 0 ? patch : null;
}

// Tool implementation
export default async function ({
    board_id,
    id,
    relevant,
    label,
    directed,
}: InferSchema<typeof schema>) {
    const res = await api.get(`/v1/boards/${board_id}`);
    const board = (await res.json()) as Board;

    let created = 0;
    let updated = 0;

    const promises = [...new Set(relevant)].map(async (target) => {
        const existing = strandBetween(board.strands, id, target);
        if (!existing) {
            created++;
            return api.post(`/v1/boards/${board_id}/strands`, {
                source: id,
                target,
                ...(label && { label }),
                ...(directed && { arrow: "end" }),
            });
        }

        const patch = restyle(existing, id, label, directed);
        if (!patch) return;
        updated++;
        return api.patch(
            `/v1/boards/${board_id}/strands/${existing.id}`,
            patch,
        );
    });

    await Promise.all(promises);

    return `Associated ${id} with ${relevant.join(", ")} (${created} new, ${updated} updated)`;
}
