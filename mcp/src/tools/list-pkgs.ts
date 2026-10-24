import type { Board } from "../types/api";

import type { ToolMetadata } from "xmcp";

import * as api from "../modules/api";

// Define the schema for tool parameters
export const schema = {};

// Define tool metadata
export const metadata: ToolMetadata = {
    name: "PKGs",
    description:
        "This tool provides you a list of personal knowledge graphs, or PKG, to which you have access to read and write, formated as `id - organization - name`. Each PKG is scoped to a theme or topic, but it is common to have a single PKG for everything. Memories within a PKG are stored in 'cards' with `id - memory`, which may take variables that are automatically populated from different sources (such as APIs) on retrieval. It is expected from you to populate this board with knowledge you gain over time, in order to preserve the data across agents and sessions, and to reduce the size of the context. You may add, retrieve, edit and delete memories as you wish, leaving a note to the user of the impacted PKG and memories.",
    annotations: {
        title: "Retrieve all the personal knowledge graphs you have access to",
        readOnlyHint: true,
        destructiveHint: false,
        idempotentHint: true,
    },
};

// Tool implementation
export default async function () {
    const res = await api.get(`/v1/boards`);
    const boards = (await res.json()) as Board[];

    const pkgs = boards.map(({ id, org, name }) => `${id} - ${org} - ${name}`);
    return pkgs.join("\n");
}
