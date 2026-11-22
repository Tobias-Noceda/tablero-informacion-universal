import type { ToolMetadata, InferSchema } from "xmcp";

import * as api from "../modules/api";
import {
    cardTypeBody,
    CardTypeError,
    FIELD_TYPES,
    undeclared,
    type CardType,
} from "../modules/card-types";

import { z } from "zod";

const fields = z
    .record(z.string().trim().min(1), z.string())
    .refine((map) => Object.keys(map).length <= 20, "At most 20 entries");

// Define the schema for tool parameters
export const schema = {
    board_id: z
        .uuid()
        .optional()
        .describe(
            "The board (PKG) the card type is for, usable by everyone on it. Leave it out for a card type of the person's own, usable on any board they edit",
        ),
    name: z
        .string()
        .trim()
        .min(1)
        .max(64)
        .describe(
            'The card type\'s name, as the cards sidebar lists it, e.g. "GitHub repository"',
        ),
    description: z
        .string()
        .trim()
        .max(200)
        .optional()
        .describe("One line on what the card shows"),
    title: z
        .string()
        .trim()
        .min(1)
        .max(127)
        .describe(
            'The title of the cards made from it, showing values as {{name}}, e.g. "{{name}}: {{stars}} stars"',
        ),
    url: z
        .string()
        .trim()
        .regex(/^https?:\/\/\S+$/, "An http(s) address")
        .describe(
            "The http(s) address the cards read, without its query string. It may use a field as $name, e.g. https://api.github.com/repos/$repo",
        ),
    response: z
        .enum(["json", "html"])
        .default("json")
        .describe("json (read with jq) or html (read with CSS selectors)"),
    query: fields
        .refine(
            (map) => Object.keys(map).length > 0,
            "At least one value to show",
        )
        .describe(
            'The values each card shows, by name: a jq expression for json (".stargazers_count") or a CSS selector for html',
        ),
    labels: fields
        .optional()
        .describe(
            'How the person sees each value\'s name, e.g. {"stars": "Stars"}; the name itself otherwise',
        ),
    queries: fields
        .optional()
        .describe(
            "Query string parameters, by name. A value may use a field as $name",
        ),
    headers: fields
        .optional()
        .describe(
            'Request headers, by name. A value may use a field as $name, e.g. {"Authorization": "$token"} with a credential field "token"',
        ),
    rate: z
        .int()
        .min(6)
        .max(86400)
        .default(600)
        .describe("How often, in seconds, the cards read the address again"),
    fields: z
        .array(
            z.object({
                name: z
                    .string()
                    .trim()
                    .min(1)
                    .max(32)
                    .describe(
                        'How the request uses it, as $name: lowercase, e.g. "repo"',
                    ),
                label: z
                    .string()
                    .trim()
                    .min(1)
                    .max(64)
                    .describe(
                        'What the person is asked, e.g. "Repository (owner/name)"',
                    ),
                type: z
                    .enum(
                        Object.keys(FIELD_TYPES) as [
                            keyof typeof FIELD_TYPES,
                            ...(keyof typeof FIELD_TYPES)[],
                        ],
                    )
                    .describe(
                        "text, number, date (YYYY-MM-DD), datetime (RFC 3339) or credential: the person picks one of their stored credentials (an API key, a token)",
                    ),
                placeholder: z
                    .string()
                    .trim()
                    .max(64)
                    .optional()
                    .describe("A hint shown in the empty field"),
                default: z
                    .string()
                    .trim()
                    .max(256)
                    .optional()
                    .describe(
                        "A value the field starts with; a field without one must be filled in",
                    ),
            }),
        )
        .max(20)
        .default([])
        .describe("What the person fills in when adding a card of this type"),
};

// Define tool metadata
export const metadata: ToolMetadata = {
    name: "create_card_type",
    description:
        "This tool lets you set up a card type for the person: a ready-made card for an API you know, which they then add from the board's cards sidebar filling in only its fields. Use a credential field for any API key or token, so each person picks their own stored credential: never write a secret in the card type. Try the request first with try_request to get the jq expressions right. Card types only make GET requests. Tell the person the card type is ready and what it will ask them for.",
    annotations: {
        title: "Set up a card type",
        readOnlyHint: false,
        destructiveHint: false,
        idempotentHint: false,
        openWorldHint: false,
    },
};

// Tool implementation
export default async function (input: InferSchema<typeof schema>) {
    let body;
    try {
        const missing = undeclared(input);
        if (missing.length) {
            return `The request uses ${missing.join(", ")}, which no field declares. Add a field for each, or write the value itself.`;
        }
        body = cardTypeBody(input);
    } catch (err) {
        if (err instanceof CardTypeError) return err.message;
        throw err;
    }

    const where = input.board_id
        ? `/v1/boards/${input.board_id}/postit-templates`
        : "/v1/users/me/postit-templates";
    let created: CardType;
    try {
        created = (await (
            await api.post(where as `/${string}`, body)
        ).json()) as CardType;
    } catch (err) {
        if (err instanceof api.ApiError && err.status === 409) {
            return "This place already has as many card types as it may: delete one first (list_card_types, delete_card_type).";
        }
        throw err;
    }

    const asks = input.fields.length
        ? `It asks for ${input.fields.map((f) => `${f.label}${f.type === "credential" ? " (a credential they pick)" : ""}`).join(", ")}.`
        : "It asks for nothing.";
    const scope = input.board_id
        ? "everyone on that board"
        : "the person, on any board they edit";
    return `Created card type "${created.name}" (${created.key}), for ${scope}: it shows up in the cards sidebar. ${asks}`;
}
