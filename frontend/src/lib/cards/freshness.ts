// When a card's data arrived: the time for today's, the day too for older.
export function formatUpdated(at: number, now: number, locale: string): string {
	const date = new Date(at);
	const today = new Date(now).toDateString() === date.toDateString();
	return new Intl.DateTimeFormat(
		locale,
		today
			? { hour: '2-digit', minute: '2-digit' }
			: { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' }
	).format(date);
}
