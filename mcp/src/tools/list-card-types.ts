import type { ToolMetadata, InferSchema } from "xmcp";

import * as api from "../modules/api";
import { describeCardType, type CardType } from "../modules/card-types";

import { z } from "zod";

// Define the schema for tool parameters
export const schema = {
    board_id: z
        .uuid()
        .optional()
        .describe(
            "A board (PKG): lists the card types usable on it (the person's own, the board's and its organization's). Leave it out for the person's own",
        ),
};

// Define tool metadata
export const metadata: ToolMetadata = {
    name: "list_card_types",
    description:
        "This tool lists the card types set up for the person or for a board, formatted as `key - name - where - what it asks for - what it shows`. Use it before creating one, to avoid a duplicate, and to find the key of one to delete.",
    annotations: {
        title: "List card types",
        readOnlyHint: true,
        destructiveHint: false,
        idempotentHint: true,
    },
};

// Tool implementation
export default async function ({ board_id }: InferSchema<typeof schema>) {
    const path = board_id
        ? `/v1/boards/${board_id}/postit-templates/usable`
        : "/v1/users/me/postit-templates";
    const types = ((await (await api.get(path as `/${string}`)).json()) ??
        []) as CardType[];
    if (types.length === 0) return "No card types yet.";
    return types.map(describeCardType).join("\n");
}
