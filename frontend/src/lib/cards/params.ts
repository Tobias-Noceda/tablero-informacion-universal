// The parameters a card is created or edited with. A blank one is left out
// rather than sent empty: the backend refuses empty values and fills a
// missing one from the card type's default.
export function filledParams(values: Record<string, string>): Record<string, string> {
	return Object.fromEntries(
		Object.entries(values)
			.map(([key, value]) => [key, value.trim()])
			.filter(([, value]) => value !== '')
	);
}
