import type { Strand, StrandArrow, UUID } from "../types/api";

// The strand between two memories, whichever way it was drawn.
export function strandBetween(strands: Strand[], a: UUID, b: UUID) {
    return strands.find(
        ({ source, target }) =>
            (source === a && target === b) || (source === b && target === a),
    );
}

// The arrow that makes the strand point away from `from`.
export function arrowFrom(strand: Strand, from: UUID): StrandArrow {
    return strand.source === from ? "end" : "start";
}

// How a strand reads from one of its memories: `→` it points to the other,
// `←` the other points to it, `↔` both ways, `—` undirected; then its label.
export function relation(strand: Strand, from: UUID) {
    const arrow = strand.arrow ?? "none";
    const direction =
        arrow === "both"
            ? "↔"
            : arrow === "none"
              ? "—"
              : arrow === arrowFrom(strand, from)
                ? "→"
                : "←";
    return strand.label
        ? `${direction} ${JSON.stringify(strand.label)}`
        : direction;
}
