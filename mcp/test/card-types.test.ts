import { describe, it } from "node:test";
import assert from "node:assert/strict";

import {
    cardTypeBody,
    CardTypeError,
    describeCardType,
    describeResponse,
    placeholder,
    undeclared,
} from "../src/modules/card-types.ts";

const repo = {
    name: "GitHub repository",
    title: "{{name}}: {{stars}} stars",
    url: "https://api.github.com/repos/$repo",
    response: "json" as const,
    query: { name: ".full_name", stars: ".stargazers_count" },
    labels: { stars: "Stars" },
    headers: { Authorization: "$token", Accept: "application/vnd.github+json" },
    rate: 600,
    fields: [
        {
            name: "repo",
            label: "Repository (owner/name)",
            type: "text" as const,
            placeholder: "sveltejs/svelte",
        },
        {
            name: "$token",
            label: "GitHub token",
            type: "credential" as const,
            default: "never kept",
        },
    ],
};

describe("cardTypeBody", () => {
    it("is a GET card type whose fields are placeholders", () => {
        assert.deepEqual(cardTypeBody(repo), {
            name: "GitHub repository",
            title: { text: "{{name}}: {{stars}} stars", vars: true },
            resource: "https://api.github.com/repos/$repo",
            request: {
                method: "GET",
                headers: {
                    Authorization: "$token",
                    Accept: "application/vnd.github+json",
                },
                queries: {},
            },
            response: "json",
            query: { name: ".full_name", stars: ".stargazers_count" },
            rate: 600,
            parameters: [
                {
                    key: "$repo",
                    label: "Repository (owner/name)",
                    type: "string",
                    placeholder: "sveltejs/svelte",
                },
                // A credential field never carries a value.
                { key: "$token", label: "GitHub token", type: "secret" },
            ],
            outputs: [
                { key: "name", label: "name" },
                { key: "stars", label: "Stars" },
            ],
        });
    });

    it("knows a title without values", () => {
        assert.equal(
            cardTypeBody({ ...repo, title: "My repo" }).title.vars,
            false,
        );
    });
});

describe("undeclared", () => {
    it("finds fields the request uses but nobody declared", () => {
        assert.deepEqual(undeclared(repo), []);
        assert.deepEqual(
            undeclared({
                ...repo,
                queries: { since: "$since" },
                fields: repo.fields.slice(0, 1),
            }),
            ["$since", "$token"],
        );
    });
});

describe("placeholder", () => {
    it("takes a name with or without $, lowercase only", () => {
        assert.equal(placeholder("repo"), "$repo");
        assert.equal(placeholder("$api_key"), "$api_key");
        assert.throws(() => placeholder("API_KEY"), CardTypeError);
        assert.throws(() => placeholder("2fa"), CardTypeError);
    });
});

describe("describeCardType", () => {
    it("says where it lives, what it asks for and shows", () => {
        assert.equal(
            describeCardType({
                key: "custom_1",
                name: "GitHub repository",
                scope: { kind: "board", owner: "b" },
                resource: "x",
                parameters: [
                    { key: "$repo", label: "Repository", type: "string" },
                    { key: "$token", label: "GitHub token", type: "secret" },
                ],
                outputs: [{ key: "stars", label: "Stars" }],
            }),
            "custom_1 - GitHub repository - this board's - asks for Repository (text), GitHub token (credential) - shows stars",
        );
    });

    it("tells one made by someone else apart", () => {
        assert.match(
            describeCardType({
                key: "k",
                name: "N",
                scope: { kind: "user", owner: "x" },
                resource: "",
                parameters: null,
                outputs: null,
                usable: false,
            }),
            /asks for nothing - not usable here/,
        );
    });
});

describe("describeResponse", () => {
    it("shows the JSON and cuts a long one", () => {
        assert.equal(describeResponse({ a: 1 }), '{\n  "a": 1\n}');
        assert.equal(describeResponse(null), "(empty response)");
        assert.match(
            describeResponse({ big: "x".repeat(7000) }),
            /… \(cut: \d+ characters in all\)$/,
        );
    });
});
