import type { BoardPostIt } from "../types/api";

import type { ToolMetadata, InferSchema } from "xmcp";

import * as api from "../modules/api";

import { z } from "zod";

// Define the schema for tool parameters
export const schema = {
    id: z.uuid().describe("The memory ID"),
    title: z
        .string()
        .trim()
        .min(1)
        .max(128)
        .optional()
        .describe("The memory title"),
    description: z
        .string()
        .trim()
        .optional()
        .describe("The memory description"),
};

// Define tool metadata
export const metadata: ToolMetadata = {
    name: "reconsolidate",
    description:
        "This tool lets you update a text memory. You should reconsolidate anything important for your task to be able to retrieve it later. Each memory should be concise, detailed and specific. Bigger concepts should be splited in smaller interconected memories.",
    annotations: {
        title: "Reconsolidate a memory",
        readOnlyHint: false,
        destructiveHint: false,
        idempotentHint: true,
    },
};

// Tool implementation
export default async function ({
    id,
    title,
    description,
}: InferSchema<typeof schema>) {
    if (title) {
        await api.patch(`/v1/post-its/${id}/title`, {
            title: { text: title, vars: false },
        });
    }

    if (description) {
        await api.patch(`/v1/post-its/${id}/settings`, {
            params: { text: description },
        });
    }

    return "Memory updated";
}
