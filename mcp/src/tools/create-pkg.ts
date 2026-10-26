import type { Board, Org } from "../types/api";

import type { ToolMetadata, InferSchema } from "xmcp";

import * as api from "../modules/api";

import { z } from "zod";

// Define the schema for tool parameters
export const schema = {
    name: z.string().trim().min(1).max(32).describe("The memory PKG name"),
    organization: z
        .string()
        .trim()
        .min(1)
        .optional()
        .describe(
            "The organization to create the PKG in, by name or id, as `list_pkgs` shows it. Omit it for a personal PKG.",
        ),
};

// Define tool metadata
export const metadata: ToolMetadata = {
    name: "create_pkg",
    description:
        "This tool provides you a way to create a personal knowledge graph, or PKG. Each PKG is a collection of memories you can organize in a canvas, joining related topics to one another, and to later search, update or delete the memories as deemed necesary.",
    annotations: {
        title: "Create a personal knowledge graphs",
        readOnlyHint: false,
        destructiveHint: false,
        idempotentHint: false,
    },
};

// Tool implementation
export default async function ({
    name,
    organization,
}: InferSchema<typeof schema>) {
    let org: string | undefined;
    if (organization) {
        const res = await api.get("/v1/orgs");
        const orgs = (await res.json()) as Org[];
        const wanted = organization.toLowerCase();
        org = orgs.find(
            (o) => o.id === organization || o.name.toLowerCase() === wanted,
        )?.id;
        if (!org)
            throw new Error(
                `You are not in an organization called "${organization}".`,
            );
    }

    const res = await api.post("/v1/boards", { name, org });
    const board = (await res.json()) as Board;
    return `Created board ${board.name} (${board.id})`;
}
