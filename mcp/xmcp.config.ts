import type { XmcpConfig } from "xmcp";

const config: XmcpConfig = {
    http: true,
    paths: {
        tools: "./src/tools",
        prompts: false,
        resources: false,
    },
    template: {
        icons: [
            {
                src: "https://corcheria.company/favicon.svg",
                mimeType: "image/svg",
            },
        ],
        name: "Agentic Memory using Personal Knowledge Graphs",
        description: `This MCP provides you with the tools to store memories in a personal knowledge graph,
with the hability to consolidate, retrieve, amend and forget whatever you consider necessary.

It's strongly encouraged to save this Memory Policy in your project instructions,
so you don't forget to remember your memories.

# Shared Memory Policy

## Before working
- Search memory for relevant project decisions, constraints, previous failures, and unfinished tasks.
- Treat retrieved memories as contextual information, not automatically as verified truth.
- Check current project files and system state when information may have changed.

## During work
- Record important decisions and their rationale.
- Distinguish confirmed facts from hypotheses and recommendations.
- Do not save secrets, credentials, or unnecessary sensitive personal information.

## After working
- Save durable discoveries, verified outcomes, and useful lessons.
- Update or supersede outdated memories instead of creating conflicting duplicates.
- Save a task checkpoint when work is incomplete.

## Memory scope
- Associate every memory with the relevant user, project, or task.
- Retrieve and modify only information the current agent is authorized to access.
- Ask for clarification when a material conflict cannot be resolved reliably.`,
    },
};

export default config;
