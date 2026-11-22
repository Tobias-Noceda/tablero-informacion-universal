// A card's settings live in its own document, which the board's live
// updates do not carry; saving them raises the card's `rev` on the board
// instead. The cards whose `rev` moved must run again. New cards fetch on
// their own when they appear.
export function changedCards(
	before: { id: string; rev?: number }[],
	after: { id: string; rev?: number }[]
): string[] {
	const previous = new Map(before.map((card) => [card.id, card.rev ?? 0]));
	return after
		.filter((card) => previous.has(card.id) && previous.get(card.id) !== (card.rev ?? 0))
		.map((card) => card.id);
}
