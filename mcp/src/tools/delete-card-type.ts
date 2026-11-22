import type { ToolMetadata, InferSchema } from "xmcp";

import * as api from "../modules/api";

import { z } from "zod";

export const schema = {
    key: z
        .string()
        .regex(
            /^custom_[0-9a-f]{32}$/,
            "A card type key, custom_ and 32 hex digits",
        )
        .describe("The card type's key, from list_card_types"),
};

export const metadata: ToolMetadata = {
    name: "delete_card_type",
    description:
        "This tool deletes a card type, e.g. one you set up by mistake. Cards already made from it keep working, but can no longer be edited. Only delete card types you set up, unless the person asks.",
    annotations: {
        title: "Delete a card type",
        readOnlyHint: false,
        destructiveHint: true,
        idempotentHint: true,
    },
};

export default async function ({ key }: InferSchema<typeof schema>) {
    await api.del(`/v1/postit-templates/${key}`);
    return `Card type ${key} deleted`;
}
