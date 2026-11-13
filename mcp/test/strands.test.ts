import { describe, it } from "node:test";
import assert from "node:assert/strict";

import type { Strand } from "../src/types/api.ts";
import { arrowFrom, relation, strandBetween } from "../src/modules/strands.ts";

const strand = (over: Partial<Strand> = {}): Strand => ({
    id: "s",
    source: "a",
    target: "b",
    ...over,
});

describe("strandBetween", () => {
    it("finds the strand whichever way it was drawn", () => {
        const strands = [
            strand({ id: "x", source: "c", target: "a" }),
            strand(),
        ];
        assert.equal(strandBetween(strands, "a", "b")?.id, "s");
        assert.equal(strandBetween(strands, "b", "a")?.id, "s");
        assert.equal(strandBetween(strands, "a", "c")?.id, "x");
        assert.equal(strandBetween(strands, "b", "c"), undefined);
    });
});

describe("arrowFrom", () => {
    it("points away from the given memory", () => {
        assert.equal(arrowFrom(strand(), "a"), "end");
        assert.equal(arrowFrom(strand(), "b"), "start");
    });
});

describe("relation", () => {
    it("reads the direction from the given memory", () => {
        assert.equal(relation(strand({ arrow: "end" }), "a"), "→");
        assert.equal(relation(strand({ arrow: "end" }), "b"), "←");
        assert.equal(relation(strand({ arrow: "start" }), "a"), "←");
        assert.equal(relation(strand({ arrow: "start" }), "b"), "→");
        assert.equal(relation(strand({ arrow: "both" }), "a"), "↔");
    });

    it("is undirected without an arrow", () => {
        assert.equal(relation(strand(), "a"), "—");
        assert.equal(relation(strand({ arrow: "none" }), "b"), "—");
    });

    it("adds the label", () => {
        assert.equal(
            relation(strand({ arrow: "end", label: "depends on" }), "a"),
            '→ "depends on"',
        );
    });
});
