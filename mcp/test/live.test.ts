import { describe, it } from "node:test";
import assert from "node:assert/strict";

import { credentialsOf, describeData, liveCardBody, requestBody } from "../src/modules/live.ts";

const memory = {
    board: "b",
    title: "Bitcoin: {{price}} USD",
    url: "https://api.coingecko.com/api/v3/simple/price",
    response: "json" as const,
    query: { price: ".bitcoin.usd" },
    queries: { ids: "bitcoin", vs_currencies: "usd" },
    headers: { "x-cg-demo-api-key": "$COINGECKO" },
    rate: 120,
};

describe("liveCardBody", () => {
    it("is a card with its own GET request", () => {
        assert.deepEqual(liveCardBody(memory), {
            board: "b",
            title: { text: "Bitcoin: {{price}} USD", vars: false },
            resource: "https://api.coingecko.com/api/v3/simple/price",
            request: {
                method: "GET",
                headers: { "x-cg-demo-api-key": "$COINGECKO" },
                queries: { ids: "bitcoin", vs_currencies: "usd" },
            },
            response: "json",
            query: { price: ".bitcoin.usd" },
            rate: 120,
        });
    });

    it("sends empty headers and queries when there are none", () => {
        const body = liveCardBody({ ...memory, headers: undefined, queries: undefined });
        assert.deepEqual(body.request, { method: "GET", headers: {}, queries: {} });
    });

    it("never names a well-known, so the backend treats it as a one-off card", () => {
        assert.equal("well_known" in liveCardBody(memory), false);
    });
});

describe("credentialsOf", () => {
    it("lists the board credentials the request names, once each", () => {
        assert.deepEqual(
            credentialsOf({
                headers: { Authorization: "$TOKEN", "x-key": "$TOKEN" },
                queries: { key: "$API_KEY", q: "plain", price: "$5" },
            }),
            ["TOKEN", "API_KEY"],
        );
    });

    it("is empty without credentials", () => {
        assert.deepEqual(credentialsOf({}), []);
    });
});

describe("describeData", () => {
    it("shows one value per line", () => {
        assert.equal(describeData({ price: 62150, coin: "bitcoin" }), "price: 62150\ncoin: bitcoin");
    });

    it("cuts long values short", () => {
        const line = describeData({ text: "a".repeat(500) });
        assert.ok(line.length < 220);
        assert.ok(line.endsWith("…"));
    });

    it("says when the card read nothing", () => {
        assert.equal(describeData({}), "(no values)");
    });
});

describe("POST memories", () => {
    const search = {
        ...memory,
        url: "https://ignav.com/api/fares/one-way",
        method: "POST" as const,
        headers: { "X-Api-Key": "$IGNAV" },
        queries: undefined,
        body: { origin: "EZE", destination: "NRT", departure_date: "2026-12-05", partner: "$PARTNER" },
    };

    it("sends the body as JSON text with the POST", () => {
        assert.deepEqual(liveCardBody(search).request, {
            method: "POST",
            headers: { "X-Api-Key": "$IGNAV" },
            queries: {},
            body: '{"origin":"EZE","destination":"NRT","departure_date":"2026-12-05","partner":"$PARTNER"}',
        });
    });

    it("never sends a body with a GET", () => {
        assert.equal(requestBody({ method: "GET", body: { a: 1 } }), "");
        assert.equal("body" in liveCardBody({ ...search, method: "GET" }).request, false);
    });

    it("lists the credentials of the headers and the body", () => {
        assert.deepEqual(credentialsOf(search), ["IGNAV", "PARTNER"]);
    });
});
